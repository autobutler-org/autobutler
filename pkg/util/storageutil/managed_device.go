package storageutil

import (
	"autobutler/pkg/util/fileutil"
	"os"
	"path/filepath"
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
		dataDir := fileutil.GetDataDirForDevice(device.MountPoint)
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
	dataDir := fileutil.GetDataDirForDevice(mountPoint)
	filesDir := filepath.Join(dataDir, "files")

	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return err
	}

	return nil
}


