package cirrusutil

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
	case "darwin": // coverage: ignore - Not run in CI
		return &DarwinDetector{}
	case "linux": // coverage: ignore - Not run in mac dev environments
		return &LinuxDetector{}
	default: // coverage: ignore - Unsupported OS
		// Unsupported platforms return empty list
		return &DarwinDetector{} // Safe fallback that returns empty list on error
	}
}
