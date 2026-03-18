package storageutil

// ManagedDevice represents a storage device that has an autobutler data directory
type ManagedDevice struct {
	Device
	DataDir   string `json:"dataDir"`   // Path to autobutler data directory on this device
	CirrusDir string `json:"cirrusDir"` // Path to cirrus subdirectory
}
