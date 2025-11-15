package storage

// Device represents a storage device with its metadata and usage information
type Device struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`         // e.g., "Internal SSD", "External USB 3.0"
	DevicePath  string            `json:"device_path"`  // e.g., "/dev/disk3s5"
	MountPoint  string            `json:"mount_point"`  // e.g., "/", "/Volumes/External"
	FileSystem  string            `json:"file_system"`  // e.g., "APFS", "ext4", "NTFS"
	TotalBytes  uint64            `json:"total_bytes"`  // Total capacity in bytes
	UsedBytes   uint64            `json:"used_bytes"`   // Used space in bytes
	AvailBytes  uint64            `json:"avail_bytes"`  // Available space in bytes
	PercentUsed int               `json:"percent_used"` // Percentage used
	IsInternal  bool              `json:"is_internal"`  // True if internal drive
	IsRemovable bool              `json:"is_removable"` // True if removable media
	IsReadOnly  bool              `json:"is_read_only"` // True if read-only
	Status      string            `json:"status"`       // "Online" or "Offline"
	Health      string            `json:"health"`       // "Good", "Excellent", etc.
	Model       string            `json:"model"`        // Device model name
	Categories  map[string]uint64 `json:"categories"`   // Breakdown by category in bytes
}
