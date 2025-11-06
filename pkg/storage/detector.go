package storage

import (
	"runtime"
)

// Detector interface for cross-platform storage detection
type Detector interface {
	DetectDevices() ([]Device, error)
	GetDeviceInfo(devicePath string) (*Device, error)
	CalculateSummary(devices []Device) Summary
}

// NewDetector creates a platform-specific storage detector
func NewDetector() Detector {
	switch runtime.GOOS {
	case "darwin":
		return &DarwinDetector{}
	case "linux":
		return &LinuxDetector{}
	default:
		return &UnsupportedDetector{}
	}
}

// UnsupportedDetector is a fallback for unsupported platforms
type UnsupportedDetector struct{}

func (d *UnsupportedDetector) DetectDevices() ([]Device, error) {
	return []Device{}, nil
}

func (d *UnsupportedDetector) GetDeviceInfo(devicePath string) (*Device, error) {
	return nil, nil
}

func (d *UnsupportedDetector) CalculateSummary(devices []Device) Summary {
	return Summary{}
}

// BytesToGB converts bytes to gigabytes
func BytesToGB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// BytesToTB converts bytes to terabytes
func BytesToTB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024 * 1024)
}
