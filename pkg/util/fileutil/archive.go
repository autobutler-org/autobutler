package fileutil

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// ListArchiveParams lists one level inside an archive. Nothing is extracted to
// disk: the storage path reads entry headers only, and the VFS path has to
// read the archive itself because a namespace has no OS path to hand a tool.
type ListArchiveParams struct {
	// Ctx bounds the VFS read.
	Ctx context.Context
	// Registry serves the listing when no serial routes past it.
	Registry vfs.Registry
	// Storage reads the archive for a device-scoped request.
	Storage *storageutil.StorageService
	// FilePath is the archive, relative to the device files directory.
	FilePath string
	// SubPath is the virtual directory inside the archive, empty for its root.
	SubPath string
	// Serial identifies the device, empty for the internal one.
	Serial string
}

// ListArchiveResult is the direct children of the requested level.
type ListArchiveResult struct {
	Entries []FileNode
}

// ListArchive returns the direct children of SubPath inside the archive at
// FilePath, as virtual paths the client passes back to list deeper.
func ListArchive(params ListArchiveParams) (ListArchiveResult, error) {
	// VFS path: only when no serial is provided.
	if params.Serial == "" {
		if fsys := FilesVFS(params.Registry); fsys != nil {
			return listArchiveVFS(params, fsys)
		}
	}

	// StorageService fallback.
	entries, err := params.Storage.ListArchiveEntries(storageutil.ListArchiveParams{
		FilePath:     params.FilePath,
		SubPath:      params.SubPath,
		DeviceSerial: params.Serial,
	})
	if err != nil {
		return ListArchiveResult{}, err
	}

	result := make([]FileNode, len(entries))
	for i, e := range entries {
		// Construct a virtual dirPath: filePath/subPath/name
		// This is the path the client must pass back as filePath to list deeper.
		virtualPath := params.FilePath
		if params.SubPath != "" {
			virtualPath = filepath.ToSlash(filepath.Join(virtualPath, params.SubPath))
		}
		dirPath := filepath.ToSlash(filepath.Join(virtualPath, e.Name))

		fileType := ""
		if !e.IsDir {
			fileType = string(storageutil.DetermineFileTypeFromPath(e.Name))
		}

		// Strip any archive extension segments from the path for display
		// but preserve the full virtual path for navigation.
		nameParts := strings.Split(e.Name, "/")
		displayName := nameParts[len(nameParts)-1]

		result[i] = FileNode{
			Name:           displayName,
			Size:           e.Size,
			CompressedSize: e.CompressedSize,
			IsDir:          e.IsDir,
			DeviceName:     "",
			DevicePath:     "",
			DirPath:        dirPath,
			FullPath:       dirPath,
			DeviceSerial:   params.Serial,
			FileType:       fileType,
		}
	}

	return ListArchiveResult{Entries: result}, nil
}

// listArchiveVFS lists an archive read out of the VFS namespace.
func listArchiveVFS(params ListArchiveParams, fsys vfs.VFS) (ListArchiveResult, error) {
	zr, err := readZipVFS(params.Ctx, fsys, params.FilePath)
	if err != nil {
		return ListArchiveResult{}, err
	}

	// Normalize subPath (no leading/trailing slash).
	normalizedSub := strings.Trim(filepath.ToSlash(params.SubPath), "/")
	prefix := ""
	if normalizedSub != "" {
		prefix = normalizedSub + "/"
	}

	seen := make(map[string]struct{})
	var result []FileNode

	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		name = strings.Trim(name, "/")
		if name == "" || name == normalizedSub {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		rel := strings.TrimPrefix(name, prefix)
		if rel == "" {
			continue
		}

		// Only the direct child (first path component).
		before, _, hasChildren := strings.Cut(rel, "/")
		childName := rel
		isDir := f.FileInfo().IsDir()
		var size int64
		var compressedSize int64
		if hasChildren {
			childName = before
			isDir = true
		} else {
			size = int64(f.UncompressedSize64)
			compressedSize = int64(f.CompressedSize64)
		}

		if _, exists := seen[childName]; exists {
			continue
		}
		seen[childName] = struct{}{}

		// Construct the virtual path for client navigation.
		virtualPath := params.FilePath
		if normalizedSub != "" {
			virtualPath = filepath.ToSlash(filepath.Join(virtualPath, normalizedSub))
		}
		dirPath := filepath.ToSlash(filepath.Join(virtualPath, childName))

		fileType := ""
		if !isDir {
			fileType = string(storageutil.DetermineFileTypeFromPath(childName))
		}

		result = append(result, FileNode{
			Name:           childName,
			Size:           size,
			CompressedSize: compressedSize,
			IsDir:          isDir,
			DeviceName:     "",
			DevicePath:     "",
			DirPath:        dirPath,
			FullPath:       dirPath,
			DeviceSerial:   params.Serial,
			FileType:       fileType,
		})
	}

	if result == nil {
		result = []FileNode{}
	}
	return ListArchiveResult{Entries: result}, nil
}

// OpenArchiveEntryParams reads one entry out of an archive without extracting
// anything to disk.
type OpenArchiveEntryParams struct {
	// Ctx bounds the VFS read.
	Ctx context.Context
	// Registry serves the read when no serial routes past it.
	Registry vfs.Registry
	// Storage reads the archive for a device-scoped request.
	Storage *storageutil.StorageService
	// ArchivePath is the archive, relative to the device files directory.
	ArchivePath string
	// EntryPath is the entry inside the archive.
	EntryPath string
	// Serial identifies the device, empty for the internal one.
	Serial string
}

// OpenArchiveEntryResult is the entry's stream. The caller closes the reader.
type OpenArchiveEntryResult struct {
	// Reader streams the decompressed entry.
	Reader io.ReadCloser
	// Size is the entry's length, negative when the archive does not say.
	Size int64
}

// OpenArchiveEntry opens a single entry inside an archive for streaming.
func OpenArchiveEntry(params OpenArchiveEntryParams) (OpenArchiveEntryResult, error) {
	// VFS path: only when no serial is provided.
	if params.Serial == "" {
		if fsys := FilesVFS(params.Registry); fsys != nil {
			return openArchiveEntryVFS(params, fsys)
		}
	}

	// StorageService fallback.
	reader, size, err := params.Storage.ReadArchiveEntry(storageutil.ReadArchiveEntryParams{
		ArchivePath:  params.ArchivePath,
		EntryPath:    params.EntryPath,
		DeviceSerial: params.Serial,
	})
	if err != nil {
		log.Printf("[files] ReadArchiveEntry failed: path=%q entry=%q err=%v", params.ArchivePath, params.EntryPath, err)
		return OpenArchiveEntryResult{}, err
	}
	return OpenArchiveEntryResult{Reader: reader, Size: size}, nil
}

// openArchiveEntryVFS finds an entry in an archive read out of the VFS namespace.
func openArchiveEntryVFS(params OpenArchiveEntryParams, fsys vfs.VFS) (OpenArchiveEntryResult, error) {
	zr, err := readZipVFS(params.Ctx, fsys, params.ArchivePath)
	if err != nil {
		return OpenArchiveEntryResult{}, err
	}

	// Normalize the requested entry path (forward slashes, no leading slash).
	normalizedEntry := strings.Trim(filepath.ToSlash(params.EntryPath), "/")

	for _, f := range zr.File {
		name := strings.Trim(filepath.ToSlash(f.Name), "/")
		if name != normalizedEntry {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return OpenArchiveEntryResult{}, fmt.Errorf("failed to open archive entry: %w", err)
		}
		return OpenArchiveEntryResult{Reader: rc, Size: int64(f.UncompressedSize64)}, nil
	}

	return OpenArchiveEntryResult{}, notFoundf("entry %q not found in archive", params.EntryPath)
}

// readZipVFS reads a whole archive out of the VFS namespace into memory. A
// namespace has no OS path, and a zip reader needs random access, so there is
// nothing to stream from.
func readZipVFS(ctx context.Context, fsys vfs.VFS, filePath string) (*zip.Reader, error) {
	r, err := fsys.Open(ctx, filePath)
	if err != nil {
		return nil, notFound(err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return zip.NewReader(bytes.NewReader(data), int64(len(data)))
}

// ExtractZipVFS extracts a .zip archive via the VFS layer.
// It reads the archive into memory, then writes each entry back via VFS.Write / VFS.MkdirAll.
func ExtractZipVFS(ctx context.Context, fsys vfs.VFS, filePath string) error {
	r, err := fsys.Open(ctx, filePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read archive: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}

	// Determine the destination directory: sibling dir named after the archive stem.
	base := filepath.Base(filePath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	destDir := path.Join(path.Dir(filePath), stem)

	if err := fsys.MkdirAll(ctx, destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Canonical Zip Slip anchor: every resolved path must begin with this prefix.
	cleanDestDir := path.Clean(destDir) + "/"

	var entryCount int
	for _, f := range zr.File {
		entryCount++
		if entryCount > storageutil.MaxArchiveEntries {
			return fmt.Errorf("archive exceeds maximum of %d entries", storageutil.MaxArchiveEntries)
		}

		// Normalize to forward-slash, clean, and strip any leading slash.
		entryName := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(f.Name)), "/")
		if entryName == "" || entryName == "." {
			continue
		}

		destPath := path.Join(destDir, entryName)

		// Zip Slip guard: the resolved destination must stay within destDir.
		// This is the canonical check CodeQL and other scanners understand.
		if !strings.HasPrefix(path.Clean(destPath)+"/", cleanDestDir) {
			continue // path traversal attempt — discard silently
		}

		if f.FileInfo().IsDir() {
			if err := fsys.MkdirAll(ctx, destPath); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			continue
		}

		// Ensure parent directory exists.
		parentDir := path.Dir(destPath)
		if err := fsys.MkdirAll(ctx, parentDir); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open archive entry %s: %w", f.Name, err)
		}

		// Apply per-entry size limit.
		limited := io.LimitReader(rc, storageutil.MaxArchiveEntryBytes+1)
		if err := fsys.Write(ctx, destPath, limited, vfs.WriteOptions{}); err != nil {
			rc.Close()
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}
		rc.Close()
	}

	return nil
}
