package fileutil

import (
	"runtime"
)

// Detector interface for cross-platform storage detection
type Detector interface {
	DetectDevices() ([]Device, error)
}

// NewDetector creates a platform-specific storage detector
func NewDetector() Detector {
	switch runtime.GOOS {
	case "darwin":
		return &DarwinDetector{}
	case "linux":
		return &LinuxDetector{}
	default:
		// Unsupported platforms return empty list
		return &DarwinDetector{} // Safe fallback that returns empty list on error
	}
}
