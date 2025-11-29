package storageutil

import (
	"os"
	"path/filepath"
	"runtime"
)

// ManagedDevice represents a storage device that has an autobutler data directory
type ManagedDevice struct {
	Device
	DataDir  string `json:"data_dir"`  // Path to autobutler data directory on this device
	FilesDir string `json:"files_dir"` // Path to files subdirectory
}

// GetManagedDevices returns all devices that have an autobutler data directory
func GetManagedDevices() ([]ManagedDevice, error) {
	detector := NewDetector()
	devices, err := detector.DetectDevices()
	if err != nil {
		return nil, err
	}

	var managedDevices []ManagedDevice
	for _, device := range devices {
		dataDir := GetDataDirForDevice(device.MountPoint)
		filesDir := filepath.Join(dataDir, "files")

		// Check if this device has an autobutler data directory
		if _, err := os.Stat(filesDir); err == nil {
			managedDevices = append(managedDevices, ManagedDevice{
				Device:   device,
				DataDir:  dataDir,
				FilesDir: filesDir,
			})
		}
	}

	return managedDevices, nil
}

// InitializeDeviceDataDir creates the autobutler data directory structure on a device
func InitializeDeviceDataDir(mountPoint string) error {
	dataDir := GetDataDirForDevice(mountPoint)
	filesDir := filepath.Join(dataDir, "files")

	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return err
	}

	return nil
}

// GetDataDirForDevice returns the data directory path for a specific device mount point
func GetDataDirForDevice(mountPoint string) string {
	// For the main system device (root filesystem or /System/Volumes/Data on macOS),
	// use the standard user-specific data directory location
	isSystemDevice := mountPoint == "/" || mountPoint == "/System/Volumes/Data"

	if isSystemDevice {
		// Use platform-specific user directories (~/Library/Application Support/Autobutler/data on macOS)
		switch runtime.GOOS {
		case "darwin":
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "/"
			}
			return filepath.Join(homeDir, "Library", "Application Support", "Autobutler", "data")
		case "linux":
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "/var/lib"
			}
			return filepath.Join(homeDir, "autobutler", "data")
		}
	}

	// For external devices, use .autobutler directory on the device itself
	return filepath.Join(mountPoint, ".autobutler", "data")
}
