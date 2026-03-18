package storageutil

// ManagedDevice represents a storage device that has an autobutler data directory
type ManagedDevice struct {
	Device
	DataDir   string `json:"dataDir"`   // Path to autobutler data directory on this device
	CirrusDir string `json:"cirrusDir"` // Path to cirrus subdirectory
}

// activeDetector is the package-level detector used by GetManagedDevices.
// It defaults to the real hardware detector and can be overridden in tests
// via SetDetectorForTesting.
var activeDetector Detector = NewDetector()

// SetDetectorForTesting replaces the package-level detector with the provided
// one and returns a cleanup function that restores the original. Use in tests:
//
//	cleanup := storageutil.SetDetectorForTesting(mockDetector)
//	defer cleanup()
func SetDetectorForTesting(d Detector) func() {
	original := activeDetector
	activeDetector = d
	return func() { activeDetector = original }
}

// FindManagedDeviceBySerial finds a managed device by its USB serial number.
// Empty serial returns first internal device.
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

// GetManagedDevices returns all devices that have an autobutler data directory.
func GetManagedDevices() ([]ManagedDevice, error) {
	devices, err := activeDetector.DetectDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	var managedDevices []ManagedDevice
	for _, device := range devices {
		dataDir := GetDataDirForDevice(device.MountPoint)
		cirrusDir, err := GetCirrusDirForDevice(device.MountPoint)
		if err != nil {
			continue
		}

		managedDevices = append(managedDevices, ManagedDevice{
			Device:    device,
			DataDir:   dataDir,
			CirrusDir: cirrusDir,
		})
	}

	return managedDevices, nil
}
