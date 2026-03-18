package storageutil

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
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
// by cross-referencing with ListUsbDevices. The UsbInfo field will be set
// if a match is found by block device path or mount point.
func (d *detector) DetectDevices() ([]Device, error) {
	devices := []Device{}

	// TODO(#663): Stop using df, which can be unreliable. Consider parsing /proc/mounts
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
		availableBytes, _ := strconv.ParseUint(fields[4], 10, 64)

		device := &Device{
			DevicePath:     devicePath,
			MountPoint:     mountPoint,
			FileSystem:     fsType,
			TotalBytes:     totalBytes,
			UsedBytes:      usedBytes,
			AvailableBytes: availableBytes,
			IsInternal:     true,
		}

		device.Name = filepath.Base(device.MountPoint)
		if device.Name == "" || device.Name == "/" {
			device.Name = "Root Volume"
		}

		device.ApplySimpleCategorization()
		devices = append(devices, *device)
		break // Only need the root volume
	}

	// Now add USB storage devices
	usbDevices, err := ListUsbDevices(true)
	if err == nil {
		for _, usb := range usbDevices {
			const storageType = "External USB"
			name := fmt.Sprintf("%s - %s", usb.GetManufacturer(), usb.GetProduct())
			mountPath := usb.GetMountPath()
			if mountPath == "" {
				// Default values, used if not mounted
				dev := Device{
					Name:           name,
					DevicePath:     "",
					MountPoint:     "",
					TotalBytes:     0,
					UsedBytes:      0,
					AvailableBytes: 0,
					IsInternal:     false,
					Model:          usb.GetProduct(),
					UsbInfo:        usb,
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
			devicePath, exists := usb.BlockDevicePath()
			if !exists {
				continue
			}
			blockSize := uint64(stat.Bsize)
			sizeBytes := stat.Blocks * blockSize
			usedBytes := (stat.Blocks - stat.Bavail) * blockSize
			availableBytes := stat.Bavail * blockSize

			device := Device{
				Name:           name,
				DevicePath:     devicePath,
				MountPoint:     mountPath,
				TotalBytes:     sizeBytes,
				UsedBytes:      usedBytes,
				AvailableBytes: availableBytes,
				IsInternal:     false,
				Model:          usb.GetProduct(),
				UsbInfo:        usb,
			}
			devices = append(devices, device)
			device.ApplySimpleCategorization()
		}
	}

	return devices, nil
}
