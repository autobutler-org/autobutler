package cirrusutil

import (
	"os"
)

// ManagedDevice represents a storage device that has an autobutler data directory
type ManagedDevice struct {
	Device
	DataDir   string `json:"data_dir"`   // Path to autobutler data directory on this device
	CirrusDir string `json:"cirrus_dir"` // Path to cirrus subdirectory
}

func FindManagedDeviceByName(name string) (*ManagedDevice, error) {
	managedDevices, err := GetManagedDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	if name != "" {
		for _, d := range managedDevices {
			if d.Name == name {
				return &d, nil
			}
		}
	}
	return nil, nil
}

// GetManagedDevices returns all devices that have an autobutler data directory
func GetManagedDevices() ([]ManagedDevice, error) {
	detector := NewDetector()
	devices, err := detector.DetectDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	var managedDevices []ManagedDevice
	for _, device := range devices {
		dataDir := GetDataDirForDevice(device.MountPoint)
		cirrusDir := ConstructCirrusDir(dataDir)

		// Check if this device has an autobutler data directory
		if _, err := os.Stat(cirrusDir); err == nil {
			managedDevices = append(managedDevices, ManagedDevice{
				Device:    device,
				DataDir:   dataDir,
				CirrusDir: cirrusDir,
			})
		}
	}

	return managedDevices, nil
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
