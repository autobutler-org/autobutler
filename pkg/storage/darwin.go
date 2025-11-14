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
		device, err := d.GetDeviceInfo(devicePath)
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
		device.PercentUsed = percentUsed
		device.MountPoint = mountPoint

		// Mark this container as seen (for deduplication in summary)
		containerID := d.getContainerID(devicePath)
		if containerID != "" {
			device.Model = containerID // Store container ID in Model for now
			seenContainers[containerID] = true
		}

		// Recalculate categories with correct values
		d.estimateCategories(device)

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

// getPhysicalDiskForContainer determines which physical disk a container is on
// For APFS containers (disk3+), returns the likely physical disk (disk0, disk1, disk2)
// For now, we assume disk3+ are synthesized APFS on disk0 (most common on Macs)
func (d *DarwinDetector) getPhysicalDiskForContainer(containerID string) string {
	// Extract disk number
	re := regexp.MustCompile(`disk(\d+)`)
	matches := re.FindStringSubmatch(containerID)
	if len(matches) > 1 {
		diskNum := matches[1]
		// disk0, disk1, disk2 are typically physical disks
		// disk3+ are typically synthesized APFS containers
		if diskNum == "0" || diskNum == "1" || diskNum == "2" {
			return containerID // Already a physical disk
		}
	}
	// Default to disk0 for synthesized containers (most Macs)
	return "disk0"
}

// getPhysicalDiskSize gets the actual physical size of a disk using diskutil
func (d *DarwinDetector) getPhysicalDiskSize(diskID string) uint64 {
	cmd := exec.Command("diskutil", "info", diskID)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	info := string(output)
	// Look for "Disk Size" which gives the physical capacity
	if sizeStr := extractValue(info, "Disk Size:"); sizeStr != "" {
		return parseSize(sizeStr)
	}
	if sizeStr := extractValue(info, "Total Size:"); sizeStr != "" {
		return parseSize(sizeStr)
	}
	return 0
}

// GetDeviceInfo retrieves detailed information about a specific device - READ ONLY
func (d *DarwinDetector) GetDeviceInfo(devicePath string) (*Device, error) {
	device := &Device{
		DevicePath: devicePath,
		Status:     "Online",
		Categories: make(map[string]uint64),
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
		device.PercentUsed = int((float64(device.UsedBytes) / float64(device.TotalBytes)) * 100)
	}

	// Determine device type and properties
	device.IsInternal = strings.Contains(strings.ToLower(info), "internal: yes")
	device.IsRemovable = strings.Contains(strings.ToLower(info), "removable media: yes") ||
		strings.Contains(strings.ToLower(info), "ejectable: yes")
	device.IsReadOnly = strings.Contains(strings.ToLower(info), "read-only volume: yes")

	// Set device type description
	device.Type = d.determineDeviceType(device, info)

	// Set default name if empty
	if device.Name == "" {
		device.Name = filepath.Base(device.MountPoint)
		if device.Name == "" || device.Name == "/" {
			device.Name = "Macintosh HD"
		}
	}

	// Check SMART status for health - READ ONLY
	device.Health = d.getHealthStatus(info)

	// Estimate categories (simplified for now)
	d.estimateCategories(device)

	return device, nil
}

// CalculateSummary calculates total storage summary from all devices
// On macOS with APFS, multiple volumes can share the same container, so we deduplicate
func (d *DarwinDetector) CalculateSummary(devices []Device) Summary {
	summary := Summary{}
	seenContainers := make(map[string]Device) // Track unique APFS containers
	physicalDisks := make(map[string]bool)    // Track which physical disks we've seen

	for _, device := range devices {
		summary.TotalDevices++

		// Check if this device is part of an APFS container we've already counted
		containerID := d.getContainerID(device.DevicePath)

		if containerID != "" {
			// Track the physical disk this container is on
			physicalDisks[d.getPhysicalDiskForContainer(containerID)] = true

			// APFS volume - only count the container once
			if existingDevice, seen := seenContainers[containerID]; seen {
				// Container already seen, use the larger values (usually the Data volume has more used space)
				if device.UsedBytes > existingDevice.UsedBytes {
					// Remove old contribution and add new
					summary.UsedBytes -= existingDevice.UsedBytes
					summary.AvailBytes -= existingDevice.AvailBytes

					summary.UsedBytes += device.UsedBytes
					summary.AvailBytes += device.AvailBytes

					seenContainers[containerID] = device
				}
				// If existing device has more used space, keep it and skip this one
			} else {
				// First time seeing this container
				summary.UsedBytes += device.UsedBytes
				summary.AvailBytes += device.AvailBytes
				seenContainers[containerID] = device
			}
		} else {
			// Not an APFS volume - track physical disk and count normally
			diskID := d.getContainerID(device.DevicePath)
			if diskID != "" {
				physicalDisks[diskID] = true
			}
			summary.TotalBytes += device.TotalBytes
			summary.UsedBytes += device.UsedBytes
			summary.AvailBytes += device.AvailBytes
		}
	}

	// For APFS systems, get actual physical disk capacity instead of df's allocatable space
	// This gives us the real disk size (e.g., 251GB) instead of container size (e.g., 239GB)
	for diskID := range physicalDisks {
		diskSize := d.getPhysicalDiskSize(diskID)
		if diskSize > 0 {
			summary.TotalBytes += diskSize
		}
	}

	// If we didn't get physical disk sizes, recalculate from available + used
	if summary.TotalBytes == 0 {
		summary.TotalBytes = summary.UsedBytes + summary.AvailBytes
	}

	summary.TotalTB = BytesToTB(summary.TotalBytes)
	summary.UsedTB = BytesToTB(summary.UsedBytes)
	summary.AvailTB = BytesToTB(summary.AvailBytes)

	return summary
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

func (d *DarwinDetector) determineDeviceType(device *Device, info string) string {
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

func (d *DarwinDetector) getHealthStatus(info string) string {
	// Check SMART status - READ ONLY
	if strings.Contains(info, "S.M.A.R.T. Status: Verified") {
		return "Excellent"
	} else if strings.Contains(info, "S.M.A.R.T. Status: Not Supported") {
		return "N/A"
	}
	return "Good"
}

func (d *DarwinDetector) estimateCategories(device *Device) {
	// TODO: Implement detailed category scanning using du command
	// For now, provide rough estimates based on common patterns

	// All categories must add up to TotalBytes (used + free = total)

	// If it's the main system volume
	if device.MountPoint == "/" || device.MountPoint == "/System/Volumes/Data" {
		// Calculate proportional estimates based on actual used space
		device.Categories["system"] = uint64(float64(device.UsedBytes) * 0.10)    // ~10% of used = system
		device.Categories["documents"] = uint64(float64(device.UsedBytes) * 0.15) // ~15% of used = documents
		device.Categories["media"] = uint64(float64(device.UsedBytes) * 0.30)     // ~30% of used = media

		// Other is the remaining used space
		categoriesTotal := device.Categories["system"] +
			device.Categories["documents"] +
			device.Categories["media"]

		if categoriesTotal <= device.UsedBytes {
			device.Categories["other"] = device.UsedBytes - categoriesTotal
		} else {
			// Fallback if calculations are off
			device.Categories["other"] = device.UsedBytes
		}
	} else {
		// For external drives, assume most is "other" usage
		device.Categories["other"] = device.UsedBytes
	}

	// Free space completes the total
	device.Categories["free"] = device.AvailBytes
}
