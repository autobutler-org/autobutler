package storageutil

type Device struct {
	Name           string            `json:"name"`
	DevicePath     string            `json:"devicePath"`        // e.g., "/dev/disk3s5"
	MountPoint     string            `json:"mountPoint"`        // e.g., "/", "/Volumes/External"
	FileSystem     string            `json:"fileSystem"`        // e.g., "APFS", "ext4", "NTFS"
	TotalBytes     uint64            `json:"totalBytes"`        // Total capacity in bytes
	UsedBytes      uint64            `json:"usedBytes"`         // Used space in bytes
	AvailableBytes uint64            `json:"availableBytes"`    // Available space in bytes
	IsInternal     bool              `json:"isInternal"`        // True if internal drive
	Model          string            `json:"model"`             // Device model name
	Categories     map[string]uint64 `json:"categories"`        // Simple heuristic breakdown in bytes
	UsbInfo        UsbDevice         `json:"usbInfo,omitempty"` // USB-specific info, if available
}

// ApplySimpleCategorization applies a simple heuristic categorization for UI display
// System volumes get a basic breakdown, external drives show as "other"
func (d *Device) ApplySimpleCategorization() {
	// TODO: This whole function is a mock and needs to be replaced with something useful later
	d.Categories = make(map[string]uint64)

	// System volume heuristic: 10% system, 20% documents, 25% media, rest is other
	if d.MountPoint == "/" || d.MountPoint == "/System/Volumes/Data" || d.MountPoint == "/home" {
		d.Categories["system"] = d.UsedBytes / 10 // 10%
		d.Categories["dents"] = d.UsedBytes / 5   // 20%
		d.Categories["media"] = d.UsedBytes / 4   // 25%

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
