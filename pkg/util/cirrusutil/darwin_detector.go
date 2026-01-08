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

var snapshotRegex = regexp.MustCompile(`/dev/disk\d+s\d+s\d+`)

// DarwinDetector implements storage detection for macOS
type DarwinDetector struct{}

// DetectDevices finds all storage devices on macOS using read-only commands
func (d *DarwinDetector) DetectDevices() ([]Device, error) {
	devices := []Device{}
	seenContainers := make(map[string]bool) // Track APFS containers to avoid double-counting

	// Use df with byte output to get mounted filesystems - READ ONLY
	cmd := exec.Command("df", "-k")
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
		if len(fields) < 6 {
			continue
		}

		devicePath := fields[0]
		mountPoint := fields[len(fields)-1]

		// Skip non-disk filesystems
		if !strings.HasPrefix(devicePath, "/dev/disk") {
			continue
		}

		// Skip system volumes we don't want to show
		if shouldSkipVolume(mountPoint) {
			continue
		}

		// Parse sizes from df (in KB, so multiply by 1024 for bytes)
		totalKB, _ := strconv.ParseUint(fields[1], 10, 64)
		usedKB, _ := strconv.ParseUint(fields[2], 10, 64)
		availKB, _ := strconv.ParseUint(fields[3], 10, 64)
		percentStr := strings.TrimSuffix(fields[4], "%")
		percentUsed, _ := strconv.Atoi(percentStr)

		// Get detailed device info
		device, err := d.getDeviceInfo(devicePath)
		if err != nil {
			continue // Skip devices we can't read
		}
		if snapshotRegex.MatchString(devicePath) {
			continue // Skip APFS snapshot devices
		}

		// Override with df values which are more accurate
		device.TotalBytes = totalKB * 1024
		device.UsedBytes = usedKB * 1024
		device.AvailBytes = availKB * 1024
		device.PercentUsed = float64(percentUsed)
		device.MountPoint = mountPoint

		// Mark this container as seen (for deduplication in summary)
		containerID := d.getContainerID(devicePath)
		if containerID != "" {
			device.Model = containerID // Store container ID in Model for now
			seenContainers[containerID] = true
		}

		// Apply simple categorization for UI
		device.ApplySimpleCategorization()

		devices = append(devices, *device)
	}

	return devices, nil
}

// getContainerID extracts the APFS container identifier from device path
// e.g., /dev/disk3s1s1 -> disk3 (the base disk)
func (d *DarwinDetector) getContainerID(devicePath string) string {
	// Extract base disk number (e.g., disk3 from /dev/disk3s1s1)
	re := regexp.MustCompile(`/dev/(disk\d+)`)
	matches := re.FindStringSubmatch(devicePath)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// getDeviceInfo retrieves detailed information about a specific device using diskutil
func (d *DarwinDetector) getDeviceInfo(devicePath string) (*Device, error) {
	device := &Device{
		DevicePath: devicePath,
	}

	// Get device info using diskutil info - READ ONLY
	cmd := exec.Command("diskutil", "info", devicePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	info := string(output)
	device.Name = extractValue(info, "Volume Name:")
	device.MountPoint = extractValue(info, "Mount Point:")
	device.FileSystem = extractValue(info, "Type \\(Bundle\\):")
	device.Model = extractValue(info, "Device / Media Name:")

	// Parse sizes
	if totalStr := extractValue(info, "Disk Size:"); totalStr != "" {
		device.TotalBytes = parseSize(totalStr)
	}
	if availStr := extractValue(info, "Volume Free Space:"); availStr != "" {
		device.AvailBytes = parseSize(availStr)
	}
	device.UsedBytes = device.TotalBytes - device.AvailBytes
	if device.TotalBytes > 0 {
		device.PercentUsed = (float64(device.UsedBytes) / float64(device.TotalBytes)) * 100
	}

	// Determine device type and properties
	device.IsInternal = strings.Contains(strings.ToLower(info), "internal: yes")
	device.IsRemovable = strings.Contains(strings.ToLower(info), "removable media: yes") ||
		strings.Contains(strings.ToLower(info), "ejectable: yes")
	device.IsReadOnly = strings.Contains(strings.ToLower(info), "read-only volume: yes")

	// Set device type description
	device.Type = determineDeviceType(device, info)

	// Set default name if empty
	if device.Name == "" {
		device.Name = filepath.Base(device.MountPoint)
		if device.Name == "" || device.Name == "/" {
			device.Name = "Macintosh HD"
		}
	}

	return device, nil
}

// Helper functions

func shouldSkipVolume(mountPoint string) bool {
	// Skip system-internal volumes
	skipPrefixes := []string{
		"/System/Volumes/VM",
		"/System/Volumes/Preboot",
		"/System/Volumes/Update",
		"/System/Volumes/xarts",
		"/System/Volumes/iSCPreboot",
		"/System/Volumes/Hardware",
		"/private/var/vm",
		"/dev",
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(mountPoint, prefix) {
			return true
		}
	}

	return false
}

func extractValue(info, key string) string {
	// Extract value after key using regex
	re := regexp.MustCompile(key + `\s+(.+)`)
	matches := re.FindStringSubmatch(info)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func parseSize(sizeStr string) uint64 {
	// Parse size strings like "245.1 GB (245107195904 Bytes)"
	re := regexp.MustCompile(`\((\d+)\s+Bytes\)`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) > 1 {
		size, _ := strconv.ParseUint(matches[1], 10, 64)
		return size
	}
	return 0
}

func determineDeviceType(device *Device, info string) string {
	var typeStr string

	// Determine connection type
	if device.IsInternal {
		if strings.Contains(info, "SSD") {
			typeStr = "Internal SSD"
		} else {
			typeStr = "Internal"
		}
	} else if device.IsRemovable {
		if strings.Contains(info, "USB") {
			typeStr = "External USB"
		} else {
			typeStr = "External"
		}
	} else {
		typeStr = "External"
	}

	// Add filesystem
	if device.FileSystem != "" {
		typeStr += " • " + device.FileSystem
	}

	return typeStr
}
