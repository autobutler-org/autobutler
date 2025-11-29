package fileutil

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
	case "/":
		return FileTypeFolder
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
	filesDir := GetFilesDir()
	stat, err := os.Stat(filepath.Join(filesDir, rootDir, file.Name()))
	if err != nil || stat == nil {
		return FileTypeGeneric // If we can't stat the file, treat it as generic
	}
	if stat.IsDir() {
		return FileTypeFolder
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
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error calculating folder size for %s: %w", dir, err)
	}
	return size, nil
}

func StatFilesInDir(dir string, deviceName string, devicePath string) ([]*DeviceFileInfo, error) {
	entries, err := os.ReadDir(dir)
	files := make([]*DeviceFileInfo, 0, len(entries))
	if err != nil {
		return nil, fmt.Errorf("error reading the directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		var fileInfo fs.FileInfo
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			folderSize, err := GetFolderSize(fullPath)
			if err != nil {
				return nil, fmt.Errorf("error getting size for folder %s: %w", entry.Name(), err)
			}
			fileInfo = NewCustomFileInfo().WithName(entry.Name()).WithSize(folderSize)
		} else {
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("error getting info for file %s: %w", entry.Name(), err)
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
			return 1 // a is a file, b is a directory
		}
		return strings.Compare(a.Name(), b.Name())
	})
	return files, nil
}

func GetAvailableSpaceInBytes(fileDir string) uint64 {
	return getAvailableSpaceInBytes(fileDir)
}

func GetDataDir() string {
	// switch on os
	switch runtime.GOOS {
	case "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Assume instead we are running as a system service, and default to a system-wide data directory.
			homeDir = "/var/lib"
		}
		return filepath.Join(homeDir, "autobutler", "data")
	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Assume instead we are running as a system service, and default to a system-wide data directory.
			homeDir = "/"
		}
		return filepath.Join(homeDir, "Library", "Application Support", "Autobutler", "data")
	default:
		panic(fmt.Sprintf("unsupported OS: %s", runtime.GOOS))
	}
}

func GetFilesDir() string {
	filesPath := filepath.Join(GetDataDir(), "files")
	if err := os.MkdirAll(filesPath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create files directory: %v", err))
	}
	return filesPath
}

// GetDeviceInfoForPath returns the device name and device path for a given file path
// This is used to populate device info even for single-device scenarios
func GetDeviceInfoForPath(path string) (deviceName string, devicePath string) {
	// Get the absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", ""
	}

	// Try to detect the device this path is on
	// This is a best-effort approach using the storage package
	// Import note: we can't import storage here due to circular dependency,
	// so we'll use a simple heuristic based on mount points

	// For now, use a simple approach: check if path starts with common mount points
	// This could be enhanced later to use actual device detection
	switch runtime.GOOS {
	case "darwin":
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
	case "linux":
		// On Linux, check if it's under /media/ or /mnt/
		if strings.HasPrefix(absPath, "/media/") {
			parts := strings.Split(absPath, "/")
			if len(parts) >= 4 {
				deviceName = parts[3] // Device name
				devicePath = filepath.Join("/media", parts[2], parts[3])
				return
			}
		} else if strings.HasPrefix(absPath, "/mnt/") {
			parts := strings.Split(absPath, "/")
			if len(parts) >= 3 {
				deviceName = parts[2] // Mount point name
				devicePath = filepath.Join("/mnt", parts[2])
				return
			}
		}
		// Default to root filesystem
		deviceName = "Root"
		devicePath = "/"
	default:
		deviceName = "Default"
		devicePath = "/"
	}

	return
}

func getAvailableSpaceInBytes(fileDir string) uint64 {
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
					continue
				}
				fileInfo = NewCustomFileInfo().WithName(entry.Name()).WithSize(folderSize)
			} else {
				info, err := entry.Info()
				if err != nil {
					continue
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
			return -1
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

// FindFileAcrossDevices searches for a file across multiple devices
// Returns the full path to the first matching file
func FindFileAcrossDevices(dirsWithDevice []DirWithDevice, relPath string) (string, error) {
	for _, dirInfo := range dirsWithDevice {
		fullPath := filepath.Join(dirInfo.Dir, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("file not found: %s", relPath)
}
