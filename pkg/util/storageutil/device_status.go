package storageutil

// DeviceStatus represents the status of a storage device, including whether it's
// enabled for Autobutler file management
type DeviceStatus struct {
	Device
	IsEnabled bool             `json:"isEnabled"`
	DataDir   string           `json:"dataDir,omitempty"`
	CirrusDir string           `json:"cirrusDir,omitempty"`
	Role      string           `json:"role"`
	DiskProbe *DiskProbeResult `json:"diskProbe,omitempty"` // nil until probed
}
