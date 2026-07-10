package vfs

import (
	"context"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

// StorageServiceVFS adapts storageutil.StorageService to the VFS interface.
// It is registered as the "files" namespace and backs the /api/v0/cirrus
// handlers during the Phase 1 migration, with no behaviour change.
type StorageServiceVFS struct {
	svc         *storageutil.StorageService
	namespaceID string
}

// NewStorageServiceVFS creates a StorageServiceVFS for the given namespace.
func NewStorageServiceVFS(svc *storageutil.StorageService, namespaceID string) *StorageServiceVFS {
	return &StorageServiceVFS{svc: svc, namespaceID: namespaceID}
}

// List returns the contents of the given directory path across all managed devices,
// deduplicating folders (same logic as the existing cirrus listFilesImpl).
func (v *StorageServiceVFS) List(_ context.Context, path string, filter *ListFilter) ([]FileInfo, error) {
	devices, err := v.svc.GetManagedDevices()
	if err != nil {
		return nil, err
	}

	var allFiles []*storageutil.DeviceFileInfo
	sawListing := false
	sawNotFound := false

	for _, device := range devices {
		fullDir, err := storageutil.SafeJoin(device.CirrusDir, path)
		if err != nil {
			continue
		}
		serial := ""
		if device.UsbInfo != nil {
			serial = device.UsbInfo.GetSerial()
		}
		files, err := storageutil.StatFilesInDir(fullDir, device.Name, device.DataDir, serial)
		if err != nil {
			if path != "" {
				sawNotFound = true
			}
			continue
		}
		sawListing = true
		allFiles = append(allFiles, files...)
	}

	if path != "" && sawNotFound && !sawListing {
		return nil, ErrNotFound
	}

	// Deduplicate directories (keep first occurrence, show all files).
	seenDirs := make(map[string]bool)
	out := make([]FileInfo, 0, len(allFiles))
	for _, f := range allFiles {
		if f.IsDir() {
			if seenDirs[f.Name()] {
				continue
			}
			seenDirs[f.Name()] = true
		}
		fi := deviceFileInfoToVFS(f, v.namespaceID, path)
		if filter != nil && filter.MimePrefix != "" && !strings.HasPrefix(fi.MimeType, filter.MimePrefix) {
			continue
		}
		out = append(out, fi)
		if filter != nil && filter.MaxResults > 0 && len(out) >= filter.MaxResults {
			break
		}
	}
	return out, nil
}

// Stat returns metadata for a single path.
func (v *StorageServiceVFS) Stat(_ context.Context, path string) (FileInfo, error) {
	result, err := v.svc.StatFile(storageutil.StatFileParams{FilePath: path})
	if err != nil {
		return FileInfo{}, ErrNotFound
	}
	mimeType := mime.TypeByExtension(filepath.Ext(result.Name))
	return FileInfo{
		Name:      result.Name,
		Path:      path,
		IsDir:     result.IsDir,
		MimeType:  mimeType,
		Namespace: v.namespaceID,
	}, nil
}

// Open returns a reader for the file at the given path.
func (v *StorageServiceVFS) Open(_ context.Context, path string) (io.ReadCloser, error) {
	result, err := v.svc.DownloadFile(storageutil.DownloadFileParams{FilePath: path})
	if err != nil {
		return nil, ErrNotFound
	}
	if result.IsFolder {
		return nil, ErrNotFound
	}
	return os.Open(result.FullPath)
}

// Write uploads a file to the vault via the StorageService.
func (v *StorageServiceVFS) Write(_ context.Context, path string, r io.Reader, opts WriteOptions) error {
	// Wrap reader in a minimal multipart-compatible form; StorageService.UploadFilesStreamed
	// expects a multipart reader. For Phase 1 the write path is not migrated through VFS
	// callers — this method exists to satisfy the interface.
	_ = path
	_ = r
	_ = opts
	return ErrWatchNotSupported // placeholder; real upload stays via StorageService directly
}

// Delete removes one or more files via the StorageService.
func (v *StorageServiceVFS) Delete(_ context.Context, path string, _ DeleteOptions) error {
	_, err := v.svc.DeleteFiles(storageutil.DeleteFilesParams{
		FilePaths: []string{path},
	})
	return err
}

// MkdirAll creates a directory (and parents) in the vault.
func (v *StorageServiceVFS) MkdirAll(_ context.Context, path string) error {
	dir, name := filepath.Split(strings.TrimRight(path, "/"))
	_, err := v.svc.CreateFolder(storageutil.CreateFolderParams{
		FolderDir:  dir,
		FolderName: name,
	})
	return err
}

// Watch is not supported by this implementation.
func (v *StorageServiceVFS) Watch(_ context.Context, _ string) (<-chan WatchEvent, error) {
	return nil, ErrWatchNotSupported
}

// deviceFileInfoToVFS converts a storageutil.DeviceFileInfo to a vfs.FileInfo.
func deviceFileInfoToVFS(f *storageutil.DeviceFileInfo, nsID, dirPath string) FileInfo {
	mimeType := mime.TypeByExtension(filepath.Ext(f.Name()))
	return FileInfo{
		Name:      f.Name(),
		Path:      filepath.Join(dirPath, f.Name()),
		Size:      f.Size(),
		IsDir:     f.IsDir(),
		MimeType:  mimeType,
		ModTime:   f.ModTime(),
		Namespace: nsID,
	}
}
