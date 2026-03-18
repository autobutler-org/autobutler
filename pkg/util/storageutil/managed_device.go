package storageutil

// ManagedDevice represents a storage device that has an autobutler data directory
type ManagedDevice struct {
	Device
	DataDir   string `json:"dataDir"`   // Path to autobutler data directory on this device
	CirrusDir string `json:"cirrusDir"` // Path to cirrus subdirectory
}

// defaultStorageService is used by the package-level function wrappers below
// so that existing callers (operations.go, etc.) continue to compile without changes.
var defaultStorageService = NewStorageService(NewDetector())

// GetManagedDevices returns all devices that have an autobutler data directory.
func GetManagedDevices() ([]ManagedDevice, error) {
	return defaultStorageService.GetManagedDevices()
}

// FindManagedDeviceBySerial finds a managed device by its USB serial number.
// Empty serial returns the first internal device.
func FindManagedDeviceBySerial(serial string) (*ManagedDevice, error) {
	return defaultStorageService.FindManagedDeviceBySerial(serial)
}
