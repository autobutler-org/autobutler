package fileutil

import (
	"io/fs"
	"time"
)

// DeviceFileInfo wraps fs.FileInfo with device information. This is the output type
// for all file functions Autobutler uses internally.
type DeviceFileInfo struct {
	fs.FileInfo
	DeviceName string // Name of the device this file is on
	DevicePath string // Mount point of the device
	FullPath   string // Full path to the file
}

// NewDeviceFileInfo is a constructor
func NewDeviceFileInfo(info fs.FileInfo, deviceName string, devicePath string, fullPath string) *DeviceFileInfo {
	return &DeviceFileInfo{
		FileInfo:   info,
		DeviceName: deviceName,
		DevicePath: devicePath,
		FullPath:   fullPath,
	}
}

// Static check to ensure DeviceFileInfo implements fs.FileInfo
var _ fs.FileInfo = (*DeviceFileInfo)(nil)

// Wrappers to embedded fs.FileInfo methods
func (d *DeviceFileInfo) Name() string       { return d.FileInfo.Name() }
func (d *DeviceFileInfo) Size() int64        { return d.FileInfo.Size() }
func (d *DeviceFileInfo) Mode() fs.FileMode  { return d.FileInfo.Mode() }
func (d *DeviceFileInfo) ModTime() time.Time { return d.FileInfo.ModTime() }
func (d *DeviceFileInfo) IsDir() bool        { return d.FileInfo.IsDir() }
func (d *DeviceFileInfo) Sys() any           { return d.FileInfo.Sys() }
