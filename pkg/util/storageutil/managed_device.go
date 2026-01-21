package storageutil

// ManagedDevice represents a storage device that has an autobutler data directory
type ManagedDevice struct {
	Device
	DataDir   string `json:"data_dir"`   // Path to autobutler data directory on this device
	CirrusDir string `json:"cirrus_dir"` // Path to cirrus subdirectory
}

// FindManagedDeviceBySerial finds a managed device by its USB serial number
// Empty serial returns first internal device
func FindManagedDeviceBySerial(serial string) (*ManagedDevice, error) {
	managedDevices, err := GetManagedDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	for _, d := range managedDevices {
		if serial == "" && d.IsInternal {
			return &d, nil
		}
		if d.UsbInfo != nil && d.UsbInfo.GetSerial() == serial {
			return &d, nil
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
		cirrusDir := GetCirrusDirForDevice(device.MountPoint)

		managedDevices = append(managedDevices, ManagedDevice{
			Device:    device,
			DataDir:   dataDir,
			CirrusDir: cirrusDir,
		})
	}

	return managedDevices, nil
}
