package storage

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LinuxDetector implements storage detection for Linux
type LinuxDetector struct{}

// DetectDevices finds all storage devices on Linux using read-only commands
func (l *LinuxDetector) DetectDevices() ([]Device, error) {
	devices := []Device{}

	// Use df to get mounted filesystems - READ ONLY
	cmd := exec.Command("df", "-B1", "--output=source,fstype,size,used,avail,pcent,target")
	output, err := cmd.Output()
	if err != nil {
		return devices, fmt.Errorf("failed to run df: %w", err)
	}

	// Parse df output
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Scan() // Skip header

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		devicePath := fields[0]
		fsType := fields[1]
		mountPoint := fields[6]

		// Skip non-physical filesystems
		if !strings.HasPrefix(devicePath, "/dev/") ||
			strings.HasPrefix(devicePath, "/dev/loop") {
			continue
		}

		// Skip system volumes we don't want to show
		if shouldSkipLinuxVolume(mountPoint) {
			continue
		}

		// Parse sizes
		totalBytes, _ := strconv.ParseUint(fields[2], 10, 64)
		usedBytes, _ := strconv.ParseUint(fields[3], 10, 64)
		availBytes, _ := strconv.ParseUint(fields[4], 10, 64)
		percentStr := strings.TrimSuffix(fields[5], "%")
		percentUsed, _ := strconv.Atoi(percentStr)

		// Create device
		device := &Device{
			DevicePath:  devicePath,
			MountPoint:  mountPoint,
			FileSystem:  fsType,
			TotalBytes:  totalBytes,
			UsedBytes:   usedBytes,
			AvailBytes:  availBytes,
			PercentUsed: percentUsed,
			Status:      "Online",
			Categories:  make(map[string]uint64),
		}

		// Get additional device info
		l.enrichDeviceInfo(device)

		devices = append(devices, *device)
	}

	return devices, nil
}

// GetDeviceInfo retrieves detailed information about a specific device - READ ONLY
func (l *LinuxDetector) GetDeviceInfo(devicePath string) (*Device, error) {
	device := &Device{
		DevicePath: devicePath,
		Status:     "Online",
		Categories: make(map[string]uint64),
	}

	// Get device info using lsblk - READ ONLY
	cmd := exec.Command("lsblk", "-no", "NAME,SIZE,TYPE,MOUNTPOINT,FSTYPE,MODEL,HOTPLUG,RO", devicePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	fields := strings.Fields(string(output))
	if len(fields) >= 5 {
		device.Model = strings.Join(fields[5:], " ")
		device.FileSystem = fields[4]
		device.IsRemovable = fields[6] == "1"
		device.IsReadOnly = fields[7] == "1"
	}

	return device, nil
}

// CalculateSummary calculates total storage summary from all devices
func (l *LinuxDetector) CalculateSummary(devices []Device) Summary {
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

// Helper functions

func shouldSkipLinuxVolume(mountPoint string) bool {
	// Skip system-internal volumes
	skipPrefixes := []string{
		"/boot",
		"/sys",
		"/proc",
		"/dev",
		"/run",
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(mountPoint, prefix) {
			return true
		}
	}

	return false
}

func (l *LinuxDetector) enrichDeviceInfo(device *Device) {
	// Set default name
	device.Name = filepath.Base(device.MountPoint)
	if device.Name == "" || device.Name == "/" {
		device.Name = "Root Volume"
	}

	// Determine if internal or external using lsblk - READ ONLY
	baseDev := extractBaseDevice(device.DevicePath)
	cmd := exec.Command("lsblk", "-no", "HOTPLUG,MODEL,TRAN", baseDev)
	output, _ := cmd.Output()

	info := string(output)
	fields := strings.Fields(info)

	if len(fields) > 0 {
		device.IsRemovable = fields[0] == "1"
		device.IsInternal = !device.IsRemovable

		if len(fields) > 1 {
			device.Model = fields[1]
		}

		// Determine device type
		device.Type = l.determineDeviceType(device, fields)
	}

	// Set health status
	device.Health = "Good"

	// Estimate categories
	l.estimateCategories(device)
}

func extractBaseDevice(devicePath string) string {
	// Extract base device from partition path
	// e.g., /dev/sda1 -> /dev/sda
	re := regexp.MustCompile(`(/dev/[a-z]+)`)
	matches := re.FindStringSubmatch(devicePath)
	if len(matches) > 1 {
		return matches[1]
	}
	return devicePath
}

func (l *LinuxDetector) determineDeviceType(device *Device, fields []string) string {
	var typeStr string

	if device.IsInternal {
		typeStr = "Internal"
	} else {
		// Check transport type
		if len(fields) > 2 {
			tran := fields[2]
			switch tran {
			case "usb":
				typeStr = "External USB"
			case "sata":
				typeStr = "External SATA"
			default:
				typeStr = "External"
			}
		} else {
			typeStr = "External"
		}
	}

	// Add filesystem
	if device.FileSystem != "" {
		typeStr += " • " + device.FileSystem
	}

	return typeStr
}

func (l *LinuxDetector) estimateCategories(device *Device) {
	// TODO: Implement detailed category scanning using du command
	// For now, provide rough estimates

	if device.MountPoint == "/" || device.MountPoint == "/home" {
		// Rough estimates for system volume
		device.Categories["system"] = uint64(float64(device.TotalBytes) * 0.10)
		device.Categories["documents"] = uint64(float64(device.UsedBytes) * 0.15)
		device.Categories["media"] = uint64(float64(device.UsedBytes) * 0.30)
		device.Categories["other"] = device.UsedBytes -
			device.Categories["system"] -
			device.Categories["documents"] -
			device.Categories["media"]
	} else {
		device.Categories["other"] = device.UsedBytes
	}

	device.Categories["free"] = device.AvailBytes
}
