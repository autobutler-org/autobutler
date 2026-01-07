package cirrusutil

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

// DetectDevices finds all storage devices on Linux using read-only commands.
//
// If a device is a USB storage device, it is enriched with USB-specific metadata
// by cross-referencing with usbutil.ListUsbDevices. The UsbInfo field will be set
// if a match is found by block device path or mount point.
func (l *LinuxDetector) DetectDevices() ([]Device, error) {
	devices := []Device{}

	// Use df to get only the root volume ("/")
	cmd := exec.Command("df", "-B1", "--output=source,fstype,size,used,avail,pcent,target")
	output, err := cmd.Output()
	if err != nil {
		return devices, fmt.Errorf("failed to run df: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Scan() // Skip header

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		mountPoint := fields[6]
		if mountPoint != "/" {
			continue
		}

		devicePath := fields[0]
		fsType := fields[1]
		totalBytes, _ := strconv.ParseUint(fields[2], 10, 64)
		usedBytes, _ := strconv.ParseUint(fields[3], 10, 64)
		availBytes, _ := strconv.ParseUint(fields[4], 10, 64)
		percentStr := strings.TrimSuffix(fields[5], "%")
		percentUsed, _ := strconv.Atoi(percentStr)

		device := &Device{
			DevicePath:  devicePath,
			MountPoint:  mountPoint,
			FileSystem:  fsType,
			TotalBytes:  totalBytes,
			UsedBytes:   usedBytes,
			AvailBytes:  availBytes,
			PercentUsed: percentUsed,
		}

		l.enrichDeviceInfo(device)
		device.ApplySimpleCategorization()
		devices = append(devices, *device)
		break // Only need the root volume
	}

	return devices, nil
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
		device.Type = determineLinuxDeviceType(device, fields)
	}
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

func determineLinuxDeviceType(device *Device, fields []string) string {
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
