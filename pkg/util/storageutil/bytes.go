package storageutil

func BytesToKB(size uint64) float64 {
	return float64(size) / 1024
}

func BytesToMB(size uint64) float64 {
	return BytesToKB(size) / 1024
}
