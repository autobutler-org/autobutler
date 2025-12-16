package cirrusutil

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
	Model       string            `json:"model"`        // Device model name
	Categories  map[string]uint64 `json:"categories"`   // Simple heuristic breakdown in bytes
}

// ApplySimpleCategorization applies a simple heuristic categorization for UI display
// System volumes get a basic breakdown, external drives show as "other"
func (d *Device) ApplySimpleCategorization() {
	// TODO: This whole function is a mock and needs to be replaced with something useful later
	d.Categories = make(map[string]uint64)

	// System volume heuristic: 10% system, 20% documents, 25% media, rest is other
	if d.MountPoint == "/" || d.MountPoint == "/System/Volumes/Data" || d.MountPoint == "/home" {
		d.Categories["system"] = d.UsedBytes / 10   // 10%
		d.Categories["documents"] = d.UsedBytes / 5 // 20%
		d.Categories["media"] = d.UsedBytes / 4     // 25%

		categorized := d.Categories["system"] + d.Categories["documents"] + d.Categories["media"]
		if categorized < d.UsedBytes {
			d.Categories["other"] = d.UsedBytes - categorized
		} else {
			d.Categories["other"] = 0 // coverage: ignore
		}
	} else {
		// External/other volumes: everything is "other"
		d.Categories["other"] = d.UsedBytes
	}
}
