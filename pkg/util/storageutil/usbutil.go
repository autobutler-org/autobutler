package storageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindUsbDeviceBySerial(serial string) (UsbDevice, error) {
	usbDevices, err := ListUsbDevices(true)
	if err != nil {
		return nil, err
	}
	for _, device := range usbDevices {
		if device.GetSerial() == serial {
			return device, nil
		}
	}
	return nil, fmt.Errorf("USB device with serial %q not found", serial)
}

// ListUsbDevices lists all USB devices under /sys/bus/usb/devices/
func ListUsbDevices(onlyStorage bool) ([]UsbDevice, error) {
	base := "/sys/bus/usb/devices/"
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var devices []UsbDevice
	for _, entry := range entries {
		devicePath := filepath.Join(base, entry.Name())
		vendor := readFileTrim(filepath.Join(devicePath, "idVendor"))
		product := readFileTrim(filepath.Join(devicePath, "idProduct"))
		if vendor == "" || product == "" {
			continue // Not a device directory
		}
		dev := &usbDevice{
			Path:         devicePath,
			VendorID:     vendor,
			ProductID:    product,
			Manufacturer: readFileTrim(filepath.Join(devicePath, "manufacturer")),
			Product:      readFileTrim(filepath.Join(devicePath, "product")),
			Serial:       readFileTrim(filepath.Join(devicePath, "serial")),
		}
		if dev.IsStorageDevice() {
			dev.UpdateStatus()
		} else if onlyStorage {
			continue
		}
		devices = append(devices, dev)
	}
	return devices, nil
}

// isDeviceMounted checks if a block device (e.g., /dev/sda) is mounted somewhere by parsing /proc/mounts.
// Returns the mount point if mounted, or empty string if not.
// isDeviceMounted checks if a block device or any of its partitions are mounted.
// Returns the mount point and the device node (e.g., /dev/sda1) if mounted, or empty string if not.
func isDeviceMounted(blockDevice string) (mountPoint string, mounted bool) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(data), "\n")
	// Check the base device (rare, usually not mounted directly)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == blockDevice {
			return fields[1], true
		}
	}
	// Check partitions (e.g., /dev/sda1, /dev/sda2)
	for partNum := 1; partNum <= 16; partNum++ {
		part := fmt.Sprintf("%s%d", blockDevice, partNum)
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			if fields[0] == part {
				return fields[1], true
			}
		}
	}
	return "", false
}

func readFileTrim(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
