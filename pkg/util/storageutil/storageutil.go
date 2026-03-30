package storageutil

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

func VideoMIMETypeFromExtension(extension string) string {
	normalizedExt := strings.TrimSpace(strings.ToLower(extension))
	if normalizedExt == "" {
		return "application/octet-stream"
	}
	if !strings.HasPrefix(normalizedExt, ".") {
		normalizedExt = "." + normalizedExt
	}

	switch normalizedExt {
	case ".mp4":
		return "video/mp4"
	case ".m4v":
		return "video/x-m4v"
	case ".webm":
		return "video/webm"
	case ".ogg", ".ogv":
		return "video/ogg"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".mpeg", ".mpg":
		return "video/mpeg"
	case ".mkv":
		return "video/x-matroska"
	case ".3gp":
		return "video/3gpp"
	case ".3g2":
		return "video/3gpp2"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	case ".ts":
		return "video/mp2t"
	default:
		return "application/octet-stream"
	}
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
	case ".zip", ".rar", ".tar", ".gz", ".tgz", ".7z":
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
	filesDir, err := GetCirrusDir()
	if err != nil {
		return FileTypeGeneric
	}
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

func StatFilesInDir(dir string, deviceName string, devicePath string, deviceSerial string) ([]*DeviceFileInfo, error) {
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
		files = append(files, NewDeviceFileInfo(fileInfo, deviceName, devicePath, fullPath, deviceSerial))
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
			deviceFileInfo := NewDeviceFileInfo(fileInfo, dirInfo.DeviceName, dirInfo.DevicePath, fullPath, dirInfo.DeviceSerial)

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
	Dir          string
	DeviceName   string
	DevicePath   string
	DeviceSerial string
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

// InitializeDeviceDataDir creates the autobutler data directory structure on a device
func InitializeDeviceDataDir(mountPoint string) error {
	dataDir := GetDataDirForDevice(mountPoint)
	cirrusDir := ConstructCirrusDir(dataDir)

	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		return err // coverage: ignore - requires filesystem permission errors
	}

	return nil
}

func FindUsbDeviceBySerial(serial string) (UsbDevice, error) {
	usbDevices, err := ListUsbDevices(true)
	if err != nil {
		return nil, err
	}
	for _, device := range usbDevices {
		if device.GetSerial() == serial {
			return device, nil
		}
	}
	return nil, fmt.Errorf("USB device with serial %q not found", serial)
}

// ListUsbDevices lists all USB devices under /sys/bus/usb/devices/
func ListUsbDevices(onlyStorage bool) ([]UsbDevice, error) {
	base := "/sys/bus/usb/devices/"
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var devices []UsbDevice
	for _, entry := range entries {
		devicePath := filepath.Join(base, entry.Name())
		vendor := readFileTrim(filepath.Join(devicePath, "idVendor"))
		product := readFileTrim(filepath.Join(devicePath, "idProduct"))
		if vendor == "" || product == "" {
			continue // Not a device directory
		}
		dev := &usbDevice{
			Path:         devicePath,
			VendorID:     vendor,
			ProductID:    product,
			Manufacturer: readFileTrim(filepath.Join(devicePath, "manufacturer")),
			Product:      readFileTrim(filepath.Join(devicePath, "product")),
			Serial:       readFileTrim(filepath.Join(devicePath, "serial")),
		}
		if dev.IsStorageDevice() {
			dev.UpdateStatus()
		} else if onlyStorage {
			continue
		}
		devices = append(devices, dev)
	}
	return devices, nil
}

// isDeviceMounted checks if a block device (e.g., /dev/sda) is mounted somewhere by parsing /proc/mounts.
// Returns the mount point if mounted, or empty string if not.
// isDeviceMounted checks if a block device or any of its partitions are mounted.
// Returns the mount point and the device node (e.g., /dev/sda1) if mounted, or empty string if not.
func isDeviceMounted(blockDevice string) (mountPoint string, mounted bool) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(data), "\n")
	// Check the base device (rare, usually not mounted directly)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == blockDevice {
			return fields[1], true
		}
	}
	// Check partitions (e.g., /dev/sda1, /dev/sda2)
	for partNum := 1; partNum <= 16; partNum++ {
		part := fmt.Sprintf("%s%d", blockDevice, partNum)
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			if fields[0] == part {
				return fields[1], true
			}
		}
	}
	return "", false
}

func readFileTrim(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
