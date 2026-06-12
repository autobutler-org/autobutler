//go:build !linux && !darwin

package storageutil

// ListUsbDevices returns an empty list on unsupported platforms.
// USB device enumeration via /sys/bus/usb/devices/ is Linux-only.
func ListUsbDevices(onlyStorage bool) ([]UsbDevice, error) {
	return []UsbDevice{}, nil
}
