package usbutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type UsbDevice interface {
	BlockDevicePath() (string, bool)
	IsMounted() (string, bool)
	IsStorageDevice() bool
	Partitions() ([]Partition, error)
}

type usbDevice struct {
	Path         string
	VendorID     string
	ProductID    string
	Manufacturer string
	Product      string
	Serial       string
}

func (u *usbDevice) BlockDevicePath() (string, bool) {
	usbDirName := filepath.Base(u.Path)
	blockDevs, _ := filepath.Glob("/sys/block/*/device")
	for _, dev := range blockDevs {
		resolved, err := filepath.EvalSymlinks(dev)
		if err != nil {
			continue
		}
		parts := strings.SplitSeq(resolved, string(os.PathSeparator))
		for part := range parts {
			if part == usbDirName {
				blockDevName := filepath.Base(filepath.Dir(dev))
				return filepath.Join("/dev", blockDevName), true
			}
		}
	}
	return "", false
}

func (u *usbDevice) IsMounted() (string, bool) {
	blockDev, exists := u.BlockDevicePath()
	if !exists {
		return "", false
	}
	return isDeviceMounted(blockDev)
}

func (u *usbDevice) IsStorageDevice() bool {
	if strings.Contains(u.Product, "Host Controller") {
		return false
	}
	_, exists := u.BlockDevicePath()
	return exists
}

// PartitionPaths returns all partitions for this USB device (e.g., /dev/sda1, /dev/sda2, ...)
func (u *usbDevice) Partitions() ([]Partition, error) {
	blockDev, exists := u.BlockDevicePath()
	if !exists {
		return nil, fmt.Errorf("block device not found")
	}
	pattern := blockDev + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var partitions []Partition
	for _, m := range matches {
		if m != blockDev {
			partitions = append(partitions, &partition{path: m})
		}
	}
	return partitions, nil
}
