package storageutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TrashDir is the directory inside each device's FilesDir where trashed files live.
const TrashDir = ".trash"

// TrashRetentionDays is how long items stay in the trash before auto-expiry.
const TrashRetentionDays = 30

// TrashEntry is the metadata stored alongside each trashed item.
type TrashEntry struct {
	OriginalPath string    `json:"originalPath"` // relative to FilesDir
	TrashedAt    time.Time `json:"trashedAt"`
}

// trashMetaFile returns the path of the JSON metadata sidecar for a trashed item.
func trashMetaFile(trashItemPath string) string {
	return trashItemPath + ".meta.json"
}

// TrashFilesParams mirrors DeleteFilesParams but moves to .trash instead of removing.
type TrashFilesParams struct {
	RootDir      string
	FilePaths    []string
	DeviceSerial string
}

// TrashFilesResult is returned on success.
type TrashFilesResult struct {
	RootDir string
}

// TrashFiles moves files/directories into the .trash folder under the device's FilesDir.
func (s *StorageService) TrashFiles(params TrashFilesParams) (*TrashFilesResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	defaultFilesDir, err := GetFilesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get files directory: %w", err)
	}

	return TrashFilesImpl(params, device, defaultFilesDir)
}

// TrashFilesImpl is the testable core of TrashFiles.
func TrashFilesImpl(params TrashFilesParams, device *ManagedDevice, defaultFilesDir string) (*TrashFilesResult, error) {
	filesDir := defaultFilesDir
	if device != nil {
		filesDir = device.FilesDir
	}

	trashRoot := filepath.Join(filesDir, TrashDir)
	if err := os.MkdirAll(trashRoot, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create trash directory: %w", err)
	}

	for _, filePath := range params.FilePaths {
		fullPath, err := safeJoin(filesDir, params.RootDir, filePath)
		if err != nil {
			return nil, fmt.Errorf("invalid file path: %w", err)
		}

		// Derive a unique trash destination (base name, de-collide with timestamp suffix).
		base := filepath.Base(fullPath)
		stamp := time.Now().UTC().Format("20060102T150405Z")
		trashDest := filepath.Join(trashRoot, stamp+"_"+base)

		if err := os.Rename(fullPath, trashDest); err != nil {
			return nil, fmt.Errorf("failed to move %s to trash: %w", filePath, err)
		}

		// Write sidecar metadata so the restore and expiry sweeper know the original path.
		relOriginal := filepath.Join(params.RootDir, filePath)
		meta := TrashEntry{
			OriginalPath: relOriginal,
			TrashedAt:    time.Now().UTC(),
		}
		metaBytes, _ := json.Marshal(meta)
		_ = os.WriteFile(trashMetaFile(trashDest), metaBytes, 0o600)
	}

	return &TrashFilesResult{RootDir: params.RootDir}, nil
}

// RestoreFileParams describes which trashed item to restore.
type RestoreFileParams struct {
	TrashName    string // base name of the item inside .trash (without .meta.json)
	DeviceSerial string
}

// RestoreFileResult is returned on success.
type RestoreFileResult struct {
	RestoredPath string // relative path the item was restored to
}

// RestoreFile moves a trashed item back to its original location.
func (s *StorageService) RestoreFile(params RestoreFileParams) (*RestoreFileResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore
	}

	defaultFilesDir, err := GetFilesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get files directory: %w", err)
	}

	return RestoreFileImpl(params, device, defaultFilesDir)
}

// RestoreFileImpl is the testable core of RestoreFile.
func RestoreFileImpl(params RestoreFileParams, device *ManagedDevice, defaultFilesDir string) (*RestoreFileResult, error) {
	filesDir := defaultFilesDir
	if device != nil {
		filesDir = device.FilesDir
	}

	trashRoot := filepath.Join(filesDir, TrashDir)
	trashDest := filepath.Join(trashRoot, params.TrashName)

	metaBytes, err := os.ReadFile(trashMetaFile(trashDest))
	if err != nil {
		return nil, fmt.Errorf("trash metadata not found for %s: %w", params.TrashName, err)
	}

	var meta TrashEntry
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("corrupt trash metadata: %w", err)
	}

	restoreTo, err := safeJoin(filesDir, meta.OriginalPath)
	if err != nil {
		return nil, fmt.Errorf("invalid original path in metadata: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(restoreTo), 0o700); err != nil {
		return nil, fmt.Errorf("failed to recreate parent directory: %w", err)
	}

	if err := os.Rename(trashDest, restoreTo); err != nil {
		return nil, fmt.Errorf("failed to restore %s: %w", params.TrashName, err)
	}

	_ = os.Remove(trashMetaFile(trashDest))

	return &RestoreFileResult{RestoredPath: meta.OriginalPath}, nil
}

// ListTrashParams lists what is in the trash for a given device.
type ListTrashParams struct {
	DeviceSerial string
}

// TrashItem describes one item in the trash.
type TrashItem struct {
	TrashName    string    `json:"trashName"`
	OriginalPath string    `json:"originalPath"`
	TrashedAt    time.Time `json:"trashedAt"`
}

// ListTrash returns all items currently in the device's trash.
func (s *StorageService) ListTrash(params ListTrashParams) ([]TrashItem, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore
	}

	defaultFilesDir, err := GetFilesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get files directory: %w", err)
	}

	return ListTrashImpl(params, device, defaultFilesDir)
}

// ListTrashImpl is the testable core of ListTrash.
func ListTrashImpl(params ListTrashParams, device *ManagedDevice, defaultFilesDir string) ([]TrashItem, error) {
	filesDir := defaultFilesDir
	if device != nil {
		filesDir = device.FilesDir
	}

	trashRoot := filepath.Join(filesDir, TrashDir)
	entries, err := os.ReadDir(trashRoot)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read trash directory: %w", err)
	}

	var items []TrashItem
	for _, entry := range entries {
		// Skip metadata sidecars — they are paired with item entries.
		name := entry.Name()
		if filepath.Ext(name) == ".json" {
			continue
		}

		metaPath := trashMetaFile(filepath.Join(trashRoot, name))
		metaBytes, err := os.ReadFile(metaPath)
		if err != nil {
			// Item without metadata — include with best-effort info.
			items = append(items, TrashItem{TrashName: name})
			continue
		}

		var meta TrashEntry
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			items = append(items, TrashItem{TrashName: name})
			continue
		}

		items = append(items, TrashItem{
			TrashName:    name,
			OriginalPath: meta.OriginalPath,
			TrashedAt:    meta.TrashedAt,
		})
	}

	return items, nil
}

// PurgeExpiredTrash deletes items that have been in the trash longer than TrashRetentionDays.
// Call this from a background ticker.
func PurgeExpiredTrashImpl(device *ManagedDevice, defaultFilesDir string) error {
	filesDir := defaultFilesDir
	if device != nil {
		filesDir = device.FilesDir
	}

	trashRoot := filepath.Join(filesDir, TrashDir)
	entries, err := os.ReadDir(trashRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read trash directory: %w", err)
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -TrashRetentionDays)

	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) == ".json" {
			continue
		}

		fullPath := filepath.Join(trashRoot, name)
		metaPath := trashMetaFile(fullPath)

		var itemTime time.Time
		metaBytes, err := os.ReadFile(metaPath)
		if err == nil {
			var meta TrashEntry
			if json.Unmarshal(metaBytes, &meta) == nil {
				itemTime = meta.TrashedAt
			}
		}

		if itemTime.IsZero() {
			// Fall back to filesystem mtime.
			info, err := entry.Info()
			if err != nil {
				continue
			}
			itemTime = info.ModTime()
		}

		if itemTime.Before(cutoff) {
			_ = os.RemoveAll(fullPath)
			_ = os.Remove(metaPath)
		}
	}

	return nil
}
