package cirrusutil

import (
	"autobutler/pkg/util/usbutil"
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// detector implements storage detection for Linux
type detector struct{}

func NewDetector() Detector {
	return &detector{}
}

// DetectDevices finds all storage devices on Linux using read-only commands.
//
// If a device is a USB storage device, it is enriched with USB-specific metadata
// by cross-referencing with usbutil.ListUsbDevices. The UsbInfo field will be set
// if a match is found by block device path or mount point.
func (d *detector) DetectDevices() ([]Device, error) {
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
			PercentUsed: float64(percentUsed),
		}

		d.enrichDeviceInfo(device)
		device.ApplySimpleCategorization()
		devices = append(devices, *device)
		break // Only need the root volume
	}

	// Now add USB storage devices
	usbDevices, err := usbutil.ListUsbDevices(true)
	if err == nil {
		for _, usb := range usbDevices {
			const storageType = "External USB"
			name := fmt.Sprintf("%s - %s", usb.GetManufacturer(), usb.GetProduct())
			mountPath := usb.GetMountPath()
			if mountPath == "" {
				// Default values, used if not mounted
				dev := Device{
					Name:        name,
					Type:        storageType,
					FileSystem:  "",
					DevicePath:  "",
					MountPoint:  "",
					TotalBytes:  0,
					UsedBytes:   0,
					AvailBytes:  0,
					PercentUsed: 0,
					IsInternal:  false,
					IsRemovable: true,
					IsReadOnly:  true,
					Model:       usb.GetProduct(),
					UsbInfo:     usb,
				}
				devices = append(devices, dev)
				continue
			}

			partitions, err := usb.Partitions()
			if err != nil || len(partitions) == 0 {
				continue
			}
			stat, err := partitions[0].Stat()
			if err != nil {
				continue
			}
			blockSize := uint64(stat.Bsize)
			sizeBytes := stat.Blocks * blockSize
			usedBytes := (stat.Blocks - stat.Bavail) * blockSize
			availableBytes := stat.Bavail * blockSize
			percentUsed := float64(usedBytes) / float64(sizeBytes) * 100

			dev := Device{
				Name:        name,
				Type:        storageType,
				FileSystem:  "",        // TODO: Not available yet
				DevicePath:  mountPath, // Not always the block device, but best available
				MountPoint:  mountPath,
				TotalBytes:  sizeBytes,
				UsedBytes:   usedBytes,
				AvailBytes:  availableBytes,
				PercentUsed: percentUsed,
				IsInternal:  false,
				IsRemovable: true,
				IsReadOnly:  false, // Assume writable
				Model:       usb.GetProduct(),
				UsbInfo:     usb,
			}
			devices = append(devices, dev)
			// Skip enrichDeviceInfo, just categorize
			dev.ApplySimpleCategorization()
		}
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

func (d *detector) enrichDeviceInfo(device *Device) {
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

	device.IsRemovable = false
	device.IsInternal = true

	if len(fields) > 0 {
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
