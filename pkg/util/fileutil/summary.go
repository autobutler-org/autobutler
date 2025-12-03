package fileutil

// Summary represents overall storage summary
type Summary struct {
	TotalDevices int     `json:"total_devices"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	AvailBytes   uint64  `json:"avail_bytes"`
	TotalTB      float64 `json:"total_tb"`
	UsedTB       float64 `json:"used_tb"`
	AvailTB      float64 `json:"avail_tb"`
}

// CalculateSummary calculates total storage summary from all devices
func CalculateSummary(devices []Device) Summary {
	summary := Summary{}

	for _, device := range devices {
		summary.TotalDevices++
		summary.TotalBytes += device.TotalBytes
		summary.UsedBytes += device.UsedBytes
		summary.AvailBytes += device.AvailBytes
	}

	summary.TotalTB = BytesToTB(summary.TotalBytes)
	summary.UsedTB = BytesToTB(summary.UsedBytes)
	summary.AvailTB = BytesToTB(summary.AvailBytes)

	return summary
}
