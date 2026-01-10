package storageutil

// Detector interface for cross-platform storage detection
type Detector interface {
	DetectDevices() ([]Device, error)
}
