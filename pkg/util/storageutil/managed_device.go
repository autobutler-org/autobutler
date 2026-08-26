package storageutil

// ManagedDevice represents a storage device that has an quark data directory
type ManagedDevice struct {
	Device
	DataDir  string `json:"dataDir"`  // Path to quark data directory on this device
	FilesDir string `json:"filesDir"` // Path to files subdirectory
}
