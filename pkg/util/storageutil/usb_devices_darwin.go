//go:build darwin

package storageutil

// ListUsbDevices returns an empty list on macOS.
// USB device enumeration via /sys/bus/usb/devices/ is Linux-only.
func ListUsbDevices(onlyStorage bool) ([]UsbDevice, error) {
	return []UsbDevice{}, nil
}
