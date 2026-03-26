package storageutil

import (
	"io/fs"
	"testing"
	"time"
)

// --- CustomFileInfo tests ---

func TestNewCustomFileInfo_ReturnsNonNil(t *testing.T) {
	info := NewCustomFileInfo()
	if info == nil {
		t.Fatal("NewCustomFileInfo() returned nil")
	}
	if info.size != 0 {
		t.Errorf("expected size 0, got %d", info.size)
	}
	if info.name != "" {
		t.Errorf("expected empty name, got %q", info.name)
	}
}

func TestCustomFileInfo_WithName_AddsTrailingSlash(t *testing.T) {
	info := NewCustomFileInfo().WithName("photos")
	if info.Name() != "photos/" {
		t.Errorf("expected %q, got %q", "photos/", info.Name())
	}
}

func TestCustomFileInfo_WithName_PreservesExistingSlash(t *testing.T) {
	info := NewCustomFileInfo().WithName("photos/")
	if info.Name() != "photos/" {
		t.Errorf("expected %q, got %q", "photos/", info.Name())
	}
}

func TestCustomFileInfo_WithSize(t *testing.T) {
	info := NewCustomFileInfo().WithSize(1024)
	if info.Size() != 1024 {
		t.Errorf("expected 1024, got %d", info.Size())
	}
}

func TestCustomFileInfo_IsDir_TrueWhenNameSet(t *testing.T) {
	info := NewCustomFileInfo().WithName("docs")
	if !info.IsDir() {
		t.Error("expected IsDir() true for name ending with /")
	}
}

func TestCustomFileInfo_Mode(t *testing.T) {
	info := NewCustomFileInfo()
	if info.Mode() != 0666 {
		t.Errorf("expected mode 0666, got %v", info.Mode())
	}
}

func TestCustomFileInfo_Sys_ReturnsNil(t *testing.T) {
	info := NewCustomFileInfo()
	if info.Sys() != nil {
		t.Error("expected Sys() to return nil")
	}
}

func TestCustomFileInfo_BuilderChaining(t *testing.T) {
	info := NewCustomFileInfo().WithName("x").WithSize(5)
	if info.Name() != "x/" {
		t.Errorf("expected %q, got %q", "x/", info.Name())
	}
	if info.Size() != 5 {
		t.Errorf("expected 5, got %d", info.Size())
	}
}

// --- DeviceFileInfo tests ---

// mockFileInfo is a minimal fs.FileInfo for testing.
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	sys     any
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() any           { return m.sys }

func TestNewDeviceFileInfo_SetsAllFields(t *testing.T) {
	mock := mockFileInfo{name: "test.txt", size: 42}
	dfi := NewDeviceFileInfo(mock, "MyDrive", "/dev/sda1", "/mnt/data/test.txt", "SN123")

	if dfi.DeviceName != "MyDrive" {
		t.Errorf("DeviceName: expected %q, got %q", "MyDrive", dfi.DeviceName)
	}
	if dfi.DevicePath != "/dev/sda1" {
		t.Errorf("DevicePath: expected %q, got %q", "/dev/sda1", dfi.DevicePath)
	}
	if dfi.FullPath != "/mnt/data/test.txt" {
		t.Errorf("FullPath: expected %q, got %q", "/mnt/data/test.txt", dfi.FullPath)
	}
	if dfi.DeviceSerial != "SN123" {
		t.Errorf("DeviceSerial: expected %q, got %q", "SN123", dfi.DeviceSerial)
	}
}

func TestDeviceFileInfo_DelegatesToEmbeddedFileInfo(t *testing.T) {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	mock := mockFileInfo{
		name:    "photo.jpg",
		size:    2048,
		mode:    0644,
		modTime: now,
		isDir:   false,
		sys:     "sentinel",
	}
	dfi := NewDeviceFileInfo(mock, "D", "/dev/sdb", "/photos/photo.jpg", "X")

	if dfi.Name() != "photo.jpg" {
		t.Errorf("Name(): expected %q, got %q", "photo.jpg", dfi.Name())
	}
	if dfi.Size() != 2048 {
		t.Errorf("Size(): expected 2048, got %d", dfi.Size())
	}
	if dfi.Mode() != 0644 {
		t.Errorf("Mode(): expected 0644, got %v", dfi.Mode())
	}
	if !dfi.ModTime().Equal(now) {
		t.Errorf("ModTime(): expected %v, got %v", now, dfi.ModTime())
	}
	if dfi.IsDir() {
		t.Error("IsDir(): expected false")
	}
	if dfi.Sys() != "sentinel" {
		t.Errorf("Sys(): expected %q, got %v", "sentinel", dfi.Sys())
	}
}

// --- CalculateSummary tests ---

func TestCalculateSummary_EmptySlice(t *testing.T) {
	s := CalculateSummary(nil)
	if s.TotalDevices != 0 || s.TotalBytes != 0 || s.UsedBytes != 0 || s.AvailBytes != 0 {
		t.Errorf("expected all zeros, got %+v", s)
	}
	if s.TotalTB != 0 || s.UsedTB != 0 || s.AvailTB != 0 {
		t.Errorf("expected TB fields zero, got %+v", s)
	}
}

func TestCalculateSummary_SingleDevice(t *testing.T) {
	devices := []Device{
		{TotalBytes: 1000, UsedBytes: 400, AvailableBytes: 600},
	}
	s := CalculateSummary(devices)
	if s.TotalDevices != 1 {
		t.Errorf("TotalDevices: expected 1, got %d", s.TotalDevices)
	}
	if s.TotalBytes != 1000 {
		t.Errorf("TotalBytes: expected 1000, got %d", s.TotalBytes)
	}
	if s.UsedBytes != 400 {
		t.Errorf("UsedBytes: expected 400, got %d", s.UsedBytes)
	}
	if s.AvailBytes != 600 {
		t.Errorf("AvailBytes: expected 600, got %d", s.AvailBytes)
	}
	if s.TotalTB != BytesToTB(1000) {
		t.Errorf("TotalTB: expected %v, got %v", BytesToTB(1000), s.TotalTB)
	}
}

func TestCalculateSummary_MultipleDevices(t *testing.T) {
	devices := []Device{
		{TotalBytes: 1000, UsedBytes: 300, AvailableBytes: 700},
		{TotalBytes: 2000, UsedBytes: 500, AvailableBytes: 1500},
		{TotalBytes: 3000, UsedBytes: 200, AvailableBytes: 2800},
	}
	s := CalculateSummary(devices)
	if s.TotalDevices != 3 {
		t.Errorf("TotalDevices: expected 3, got %d", s.TotalDevices)
	}
	if s.TotalBytes != 6000 {
		t.Errorf("TotalBytes: expected 6000, got %d", s.TotalBytes)
	}
	if s.UsedBytes != 1000 {
		t.Errorf("UsedBytes: expected 1000, got %d", s.UsedBytes)
	}
	if s.AvailBytes != 5000 {
		t.Errorf("AvailBytes: expected 5000, got %d", s.AvailBytes)
	}
	if s.TotalTB != BytesToTB(6000) {
		t.Errorf("TotalTB: expected %v, got %v", BytesToTB(6000), s.TotalTB)
	}
	if s.UsedTB != BytesToTB(1000) {
		t.Errorf("UsedTB: expected %v, got %v", BytesToTB(1000), s.UsedTB)
	}
	if s.AvailTB != BytesToTB(5000) {
		t.Errorf("AvailTB: expected %v, got %v", BytesToTB(5000), s.AvailTB)
	}
}
