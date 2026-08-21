package vfs

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
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

// serialSet builds a set from a slice for O(1) lookup.
func serialSet(serials []string) map[string]bool {
	set := make(map[string]bool, len(serials))
	for _, s := range serials {
		set[s] = true
	}
	return set
}

// List returns the contents of the given directory path across all managed devices,
// deduplicating folders (same logic as the existing cirrus listFilesImpl).
func (v *StorageServiceVFS) List(_ context.Context, path string, filter *ListFilter) ([]FileInfo, error) {
	devices, err := v.svc.GetManagedDevices()
	if err != nil {
		return nil, err
	}

	// Build serial filter set once (empty map = no filter).
	var allowedSerials map[string]bool
	if filter != nil && len(filter.SerialFilter) > 0 {
		allowedSerials = serialSet(filter.SerialFilter)
	}

	var allFiles []*storageutil.DeviceFileInfo
	sawListing := false
	sawNotFound := false

	for _, device := range devices {
		serial := ""
		if device.UsbInfo != nil {
			serial = device.UsbInfo.GetSerial()
		}
		// Apply serial filter.
		if allowedSerials != nil && !allowedSerials[serial] {
			continue
		}
		fullDir, err := storageutil.SafeJoin(device.CirrusDir, path)
		if err != nil {
			continue
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

// cirrusDir resolves the base directory for this namespace, preferring the
// managed device's cirrus directory over the default. StatFile, DownloadFile,
// and DeleteFiles all resolve this way internally; this exists so the paths
// derived directly in this file agree with them.
func (v *StorageServiceVFS) cirrusDir() (string, error) {
	device, err := v.svc.FindManagedDeviceBySerial("")
	if err != nil {
		return "", err
	}
	if device != nil && device.CirrusDir != "" {
		return device.CirrusDir, nil
	}
	return storageutil.GetCirrusDir()
}

// mimeTypeForName returns the MIME type for a file name. Image formats
// (including HEIC/HEIF/TIFF/BMP, which Go's stdlib mime package doesn't
// recognize — see #1567) are resolved via storageutil's own extension table
// rather than mime.TypeByExtension, which returns "" for them and silently
// disables server-side JPEG conversion for image previews.
func mimeTypeForName(name string) string {
	ext := filepath.Ext(name)
	if storageutil.DetermineFileTypeFromPath(name) == storageutil.FileTypeImage {
		return storageutil.ImageMIMETypeFromExtension(ext)
	}
	return mime.TypeByExtension(ext)
}

// Stat returns metadata for a single path.
func (v *StorageServiceVFS) Stat(_ context.Context, path string) (FileInfo, error) {
	result, err := v.svc.StatFile(storageutil.StatFileParams{FilePath: path})
	if err != nil {
		return FileInfo{}, ErrNotFound
	}
	mimeType := mimeTypeForName(result.Name)
	return FileInfo{
		Name:      result.Name,
		Path:      path,
		IsDir:     result.IsDir,
		Size:      result.Size,
		ModTime:   result.ModTime,
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
	// Use the path DownloadFile already resolved. It accounts for the managed
	// device's cirrus directory, which may differ from the default one — see
	// #1538, where re-deriving from GetCirrusDir() here made Stat and Open
	// disagree and downloads returned an empty body.
	//
	// DownloadFile validates via safeJoin internally; Clean again so static
	// analysers (CodeQL go/path-injection) can follow the traversal guard
	// rather than seeing tainted data reach os.Open.
	safePath := filepath.Clean(result.FullPath)
	f, err := os.Open(safePath) //nolint:gosec // path validated by DownloadFile's safeJoin + Clean
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return f, nil
}

// Write writes a file into the cirrus directory of the managed device that
// backs this namespace, falling back to the default cirrus directory when no
// device is present. Resolving the same way Stat and Open do keeps a written
// file findable by a subsequent read (#1538).
func (v *StorageServiceVFS) Write(_ context.Context, path string, r io.Reader, opts WriteOptions) error {
	cirrusDir, err := v.cirrusDir()
	if err != nil {
		return err
	}
	// filepath.Clean before SafeJoin so static analysers (CodeQL go/path-injection)
	// can follow the traversal guard rather than seeing tainted data reach os.Create.
	safePath, err := storageutil.SafeJoin(cirrusDir, filepath.Clean(path))
	if err != nil {
		return ErrPermissionDenied
	}
	safePath = filepath.Clean(safePath)
	if opts.IfNoneMatch == "*" {
		if _, statErr := os.Stat(safePath); statErr == nil {
			return ErrConflict
		}
	}
	if err := os.MkdirAll(filepath.Dir(safePath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(safePath) //nolint:gosec // path already validated by SafeJoin + filepath.Clean
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
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

// Move renames src to dst via the StorageService.
func (v *StorageServiceVFS) Move(_ context.Context, src, dst string) error {
	_, err := v.svc.MoveFile(storageutil.MoveFileParams{
		OldFilePath: src,
		NewFilePath: dst,
	})
	return err
}

// Watch is not supported by this implementation.
func (v *StorageServiceVFS) Watch(_ context.Context, _ string) (<-chan WatchEvent, error) {
	return nil, ErrWatchNotSupported
}

// deviceFileInfoToVFS converts a storageutil.DeviceFileInfo to a vfs.FileInfo.
func deviceFileInfoToVFS(f *storageutil.DeviceFileInfo, nsID, dirPath string) FileInfo {
	mimeType := mimeTypeForName(f.Name())
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
