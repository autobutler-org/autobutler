package storageutil

func BytesToKB(size uint64) float64 {
	return float64(size) / 1024
}

func BytesToMB(size uint64) float64 {
	return BytesToKB(size) / 1024
}

func BytesToGB(size uint64) float64 {
	return BytesToMB(size) / 1024
}

func BytesToTB(size uint64) float64 {
	return BytesToGB(size) / 1024
}

func KBToBytes(size float64) uint64 {
	return uint64(size * 1024)
}

func MBToBytes(size float64) uint64 {
	return uint64(KBToBytes(size) * 1024)
}

func GBToBytes(size float64) uint64 {
	return uint64(MBToBytes(size) * 1024)
}

func TBToBytes(size float64) uint64 {
	return uint64(GBToBytes(size) * 1024)
}
