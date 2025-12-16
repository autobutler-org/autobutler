package cirrusutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"golang.org/x/sys/unix"
)

type FileType string

const (
	FileTypeDocx      FileType = "docx"
	FileTypeEpub      FileType = "epub"
	FileTypeFolder    FileType = "folder"
	FileTypeGeneric   FileType = "generic"
	FileTypeImage     FileType = "image"
	FileTypePDF       FileType = "pdf"
	FileTypeSlideshow FileType = "slideshow"
	FileTypeVideo     FileType = "video"
	FileTypeSpacer    FileType = "spacer"
	FileTypeArchive   FileType = "archive"
)

func BytesToKB(size uint64) float64 {
	return float64(size) / 1024
}

func BytesToMB(size uint64) float64 {
	return BytesToKB(size) / 1024
}

func BytesToGB(size uint64) float64 {
	return BytesToMB(size) / 1024
}

func BytesToTB(size uint64) float64 {
	return BytesToGB(size) / 1024
}

func KBToBytes(size float64) uint64 {
	return uint64(size * 1024)
}

func MBToBytes(size float64) uint64 {
	return uint64(KBToBytes(size) * 1024)
}

func GBToBytes(size float64) uint64 {
	return uint64(MBToBytes(size) * 1024)
}

func TBToBytes(size float64) uint64 {
	return uint64(GBToBytes(size) * 1024)
}

func DetermineFileTypeFromPath(filePath string) FileType {
	// Empty string or "/" represents a folder
	if filePath == "" || filePath == "/" {
		return FileTypeFolder
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return FileTypePDF
	case ".pptx", ".ppt":
		return FileTypeSlideshow
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".heic", ".heif", ".webp", ".bmp", ".tiff", ".tif", ".avif":
		return FileTypeImage
	case ".mp4", ".m4v", ".webm", ".ogg", ".avi", ".mov":
		return FileTypeVideo
	case ".epub":
		return FileTypeEpub
	case ".docx":
		return FileTypeDocx
	case ".zip", ".rar", ".tar", ".gz", ".7z":
		return FileTypeArchive
	default:
		stat, err := os.Stat(filePath)
		if err == nil && stat != nil {
			if stat.IsDir() {
				return FileTypeFolder
			}
		}
		return FileTypeGeneric
	}
}

func DetermineFileType(rootDir string, file *DeviceFileInfo) FileType {
	if file == nil {
		return FileTypeSpacer
	}
	if file.IsDir() {
		return FileTypeFolder
	}
	filesDir := GetCirrusDir()
	stat, err := os.Stat(filepath.Join(filesDir, rootDir, file.Name()))
	if err != nil || stat == nil {
		return FileTypeGeneric // If we can't stat the file, treat it as generic
	}
	return DetermineFileTypeFromPath(file.Name())
}

func SizeBytesToString(size_bytes int64) string {
	if size_bytes < 1024 {
		return fmt.Sprintf("%d B", size_bytes)
	} else if size_bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size_bytes)/1024)
	} else if size_bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size_bytes)/(1024*1024))
	} else if size_bytes < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(size_bytes)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.1f TB", float64(size_bytes)/(1024*1024*1024*1024))
	}
}

func GetFolderSize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(_ string, info fs.FileInfo, err error) error {
		if err != nil {
			return err // coverage: ignore - requires filesystem permission errors during walk
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error calculating folder size for %s: %w", dir, err) // coverage: ignore - requires filesystem permission errors during walk
	}
	return size, nil
}

func StatFilesInDir(dir string, deviceName string, devicePath string) ([]*DeviceFileInfo, error) {
	entries, err := os.ReadDir(dir)
	files := make([]*DeviceFileInfo, 0, len(entries))
	if err != nil {
		return nil, fmt.Errorf("error reading the directory %s: %w", dir, err) // coverage: ignore - requires filesystem permission errors
	}
	for _, entry := range entries {
		var fileInfo fs.FileInfo
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			folderSize, err := GetFolderSize(fullPath)
			if err != nil {
				return nil, fmt.Errorf("error getting size for folder %s: %w", entry.Name(), err) // coverage: ignore - requires filesystem errors during folder traversal
			}
			fileInfo = NewCustomFileInfo().WithName(entry.Name()).WithSize(folderSize)
		} else {
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("error getting info for file %s: %w", entry.Name(), err) // coverage: ignore - requires filesystem errors on stat
			}
			fileInfo = info
		}
		// Wrap in DeviceFileInfo with device info
		files = append(files, NewDeviceFileInfo(fileInfo, deviceName, devicePath, fullPath))
	}
	// Sort files by directory first, then by name
	slices.SortFunc(files, func(a, b *DeviceFileInfo) int {
		if a.IsDir() && !b.IsDir() {
			return -1 // a is a directory, b is a file
		} else if !a.IsDir() && b.IsDir() {
			return 1 // coverage: ignore - a is a file, b is a directory
		}
		return strings.Compare(a.Name(), b.Name())
	})
	return files, nil
}

// GetDataDirForDevice returns the data directory path for a specific device mount point
func GetDataDirForDevice(mountPoint string) string {
	// For the main system device (root filesystem or /System/Volumes/Data on macOS),
	// use the standard user-specific data directory location
	isSystemDevice := mountPoint == "/" || mountPoint == "/System/Volumes/Data"

	if isSystemDevice {
		// Use platform-specific user directories (~/Library/Application Support/Autobutler/data on macOS)
		switch runtime.GOOS {
		case "darwin": // coverage: ignore - Not run in CI
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "/" // coverage: ignore - requires UserHomeDir to fail
			}
			return filepath.Join(homeDir, "Library", "Application Support", "Autobutler", "data")
		case "linux": // coverage: ignore - Not run in mac dev environments
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "/var/lib" // coverage: ignore - requires UserHomeDir to fail
			}
			return filepath.Join(homeDir, "autobutler", "data") // coverage: ignore
		}
	}

	// For external devices, use .autobutler directory on the device itself
	return filepath.Join(mountPoint, ".autobutler", "data")
}

func GetDataDir() string {
	// Get data directory for the system device
	switch runtime.GOOS {
	case "darwin": // coverage: ignore - Not run in CI
		// On macOS, the system mount point can be / or /System/Volumes/Data
		// Use / as the canonical reference
		return GetDataDirForDevice("/")
	case "linux": // coverage: ignore - Not run in mac dev environments
		return GetDataDirForDevice("/")
	default:
		panic(fmt.Sprintf("unsupported OS: %s", runtime.GOOS)) // coverage: ignore - panic on unsupported OS
	}
}

// GetNonConflictingPath returns a file path that doesn't conflict with existing files.
// If the target path already exists, it appends _(n) before the file extension,
// incrementing n until a non-existent path is found.
// For example: file.txt -> file_(1).txt -> file_(2).txt, etc.
func GetNonConflictingPath(targetPath string) string {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// File doesn't exist, return the target path as-is
		return targetPath
	}

	// File exists, need to find a non-conflicting name
	ext := filepath.Ext(targetPath)
	nameWithoutExt := targetPath[:len(targetPath)-len(ext)]
	dir := filepath.Dir(targetPath)

	i := 1
	for {
		newPath := filepath.Join(dir, fmt.Sprintf("%s_(%d)%s", filepath.Base(nameWithoutExt), i, ext))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		i++
	}
}

func SetupCirrusDir() error {
	// Check if the cirrus dir exists
	// If cirrus dir exists, check if legacy files dir exists
	// if so, move the contents of legacy files dir to cirrus dir, then delete legacy files dir
	cirrusDir := ConstructCirrusDir(GetDataDir())
	legacyFilesDir := filepath.Join(GetDataDir(), "files")

	if _, err := os.Stat(cirrusDir); os.IsNotExist(err) {
		// Cirrus dir does not exist, create it
		if err := os.MkdirAll(cirrusDir, 0755); err != nil {
			return fmt.Errorf("failed to create cirrus directory: %w", err)
		}
	}

	if _, err := os.Stat(legacyFilesDir); err == nil {
		// Legacy files dir exists, move contents to cirrus dir
		entries, err := os.ReadDir(legacyFilesDir)
		if err != nil {
			return fmt.Errorf("failed to read legacy files directory: %w", err)
		}
		for _, entry := range entries {
			oldPath := filepath.Join(legacyFilesDir, entry.Name())
			targetPath := filepath.Join(cirrusDir, entry.Name())
			// Use GetNonConflictingPath to handle naming conflicts
			newPath := GetNonConflictingPath(targetPath)
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to move file %s to cirrus directory: %w", entry.Name(), err)
			}
		}
		// Delete legacy files dir
		if err := os.RemoveAll(legacyFilesDir); err != nil {
			return fmt.Errorf("failed to delete legacy files directory: %w", err)
		}
	}

	return nil
}

func ConstructCirrusDir(dataDir string) string {
	return filepath.Join(dataDir, "cirrus")
}

func GetCirrusDir() string {
	cirrusPath := ConstructCirrusDir(GetDataDir())
	if err := os.MkdirAll(cirrusPath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create cirrus directory: %v", err)) // coverage: ignore - panic on filesystem error
	}
	return cirrusPath
}

// GetDeviceInfoForPath returns the device name and device path for a given file path
// This is used to populate device info even for single-device scenarios
func GetDeviceInfoForPath(path string) (deviceName string, devicePath string) {
	// Get the absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "" // coverage: ignore - requires filepath.Abs to fail
	}

	// Try to detect the device this path is on
	// This is a best-effort approach using the storage package
	// Import note: we can't import storage here due to circular dependency,
	// so we'll use a simple heuristic based on mount points

	// For now, use a simple approach: check if path starts with common mount points
	// This could be enhanced later to use actual device detection
	switch runtime.GOOS {
	case "darwin": // coverage: ignore - Not run in CI
		// On macOS, check if it's under /Volumes/
		if strings.HasPrefix(absPath, "/Volumes/") {
			parts := strings.Split(absPath, "/")
			if len(parts) >= 3 {
				deviceName = parts[2] // Volume name
				devicePath = filepath.Join("/Volumes", parts[2])
				return
			}
		}
		// Default to "Macintosh HD" for main drive
		deviceName = "Macintosh HD"
		devicePath = "/"
	case "linux": // coverage: ignore - Not run in mac dev environments
		// On Linux, check if it's under /media/ or /mnt/
		if strings.HasPrefix(absPath, "/media/") { // coverage: ignore
			parts := strings.Split(absPath, "/")
			if len(parts) >= 4 { // coverage: ignore
				deviceName = parts[3] // Device name
				devicePath = filepath.Join("/media", parts[2], parts[3])
				return
			}
		} else if strings.HasPrefix(absPath, "/mnt/") { // coverage: ignore
			parts := strings.Split(absPath, "/")
			if len(parts) >= 3 { // coverage: ignore
				deviceName = parts[2] // Mount point name
				devicePath = filepath.Join("/mnt", parts[2])
				return
			}
		}
		// Default to root filesystem
		deviceName = "Root" // coverage: ignore
		devicePath = "/"    // coverage: ignore
	default: // coverage: ignore - unsupported OS
		deviceName = "Default"
		devicePath = "/"
	}

	return
}

func GetAvailableSpaceInBytes(fileDir string) uint64 {
	var stat unix.Statfs_t
	unix.Statfs(fileDir, &stat)
	return stat.Bavail * uint64(stat.Bsize)
}

// StatFilesInMultipleDirs reads files from multiple directories and merges them
// into a unified view, preserving device information for each file
func StatFilesInMultipleDirs(dirsWithDeviceInfo []DirWithDevice) ([]*DeviceFileInfo, error) {
	// Map to merge files by name
	fileMap := make(map[string][]*DeviceFileInfo)

	// Read files from each device
	for _, dirInfo := range dirsWithDeviceInfo {
		entries, err := os.ReadDir(dirInfo.Dir)
		if err != nil {
			// Skip directories that don't exist or can't be read
			continue
		}

		for _, entry := range entries {
			var fileInfo fs.FileInfo
			fullPath := filepath.Join(dirInfo.Dir, entry.Name())

			if entry.IsDir() {
				folderSize, err := GetFolderSize(fullPath)
				if err != nil {
					continue // coverage: ignore - requires filesystem errors during folder size calculation
				}
				fileInfo = NewCustomFileInfo().WithName(entry.Name()).WithSize(folderSize)
			} else {
				info, err := entry.Info()
				if err != nil {
					continue // coverage: ignore - requires filesystem errors on stat
				}
				fileInfo = info
			}

			// Wrap with device information
			deviceFileInfo := NewDeviceFileInfo(fileInfo, dirInfo.DeviceName, dirInfo.DevicePath, fullPath)

			// Add to map (multiple devices may have same file name)
			fileMap[entry.Name()] = append(fileMap[entry.Name()], deviceFileInfo)
		}
	}

	// Flatten the map to a slice
	var files []*DeviceFileInfo
	for _, fileList := range fileMap {
		files = append(files, fileList...)
	}

	// Sort files by directory first, then by name
	slices.SortFunc(files, func(a, b *DeviceFileInfo) int {
		if a.IsDir() && !b.IsDir() {
			return -1 // coverage: ignore - a is a directory, b is a file
		} else if !a.IsDir() && b.IsDir() {
			return 1
		}
		return strings.Compare(a.Name(), b.Name())
	})

	return files, nil
}

// DirWithDevice associates a directory path with its device information
type DirWithDevice struct {
	Dir        string
	DeviceName string
	DevicePath string
}

// DoesFileExist checks if a file exists at the given path
// Returns true if the file exists, false otherwise
func DoesFileExist(fullPath string) bool {
	_, err := os.Stat(fullPath)
	return err == nil
}

// FindFileAcrossDevices searches for a file across multiple devices
// Returns the full path to the first matching file
func FindFileAcrossDevices(dirsWithDevice []DirWithDevice, relPath string) (string, error) {
	for _, dirInfo := range dirsWithDevice {
		fullPath := filepath.Join(dirInfo.Dir, relPath)
		if DoesFileExist(fullPath) {
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("file not found: %s", relPath)
}
