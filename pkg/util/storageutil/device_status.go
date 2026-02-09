package storageutil

// DeviceStatus represents the status of a storage device, including whether it's
// enabled for Autobutler file management
type DeviceStatus struct {
	Device
	IsEnabled bool   `json:"isEnabled"`
	DataDir   string `json:"dataDir,omitempty"`
	CirrusDir string `json:"cirrusDir,omitempty"`
}

// GetDeviceStatuses returns all detected devices with their enable status
func GetDeviceStatuses() ([]*DeviceStatus, error) {
	// Detect all storage devices
	detector := NewDetector()
	devices, err := detector.DetectDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection to fail
	}

	// Get managed devices to check which are enabled
	managedDevices, err := GetManagedDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires filesystem errors reading managed devices
	}

	// Create a map of enabled devices
	enabledMap := make(map[string]ManagedDevice)
	for _, md := range managedDevices {
		enabledMap[md.MountPoint] = md
	}

	// Build status list
	var statuses []*DeviceStatus
	for _, device := range devices {
		// Internal devices are enabled by default
		isEnabled := device.IsInternal
		dataDir := ""
		cirrusDir := ""

		// Check if device is in managed devices (explicitly enabled)
		if md, exists := enabledMap[device.MountPoint]; exists {
			isEnabled = true
			dataDir = md.DataDir
			cirrusDir = md.CirrusDir
		}

		status := &DeviceStatus{
			Device:    device,
			IsEnabled: isEnabled,
			DataDir:   dataDir,
			CirrusDir: cirrusDir,
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
