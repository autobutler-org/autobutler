package storage

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
