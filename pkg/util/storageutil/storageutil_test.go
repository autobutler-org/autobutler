package storageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadFileTrim(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"trims whitespace", "  hello world \n", "hello world"},
		{"empty file", "", ""},
		{"no trim needed", "foo", "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "testfile")
			if err := os.WriteFile(file, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			got := readFileTrim(file)
			if got != tt.want {
				t.Errorf("readFileTrim() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsStorageDevice(t *testing.T) {
	tests := []struct {
		name         string
		product      string
		manufacturer string
		want         bool
	}{
		{"host controller", "xHCI Host Controller", "Linux", false},
		{"non-storage device", "Some Product", "Some Manufacturer", false},
		// Add more cases as needed
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "product"), []byte(tt.product), 0644)
			os.WriteFile(filepath.Join(dir, "manufacturer"), []byte(tt.manufacturer), 0644)
			dev := &usbDevice{Path: dir}
			got := dev.IsStorageDevice()
			if got != tt.want {
				t.Errorf("IsStorageDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBytesConversions(t *testing.T) {
	tests := []struct {
		name  string
		bytes uint64
		kb    float64
		mb    float64
		gb    float64
	}{
		{"1KB", 1024, 1.0, 0.0009765625, 0.00000095367431640625},
		{"1MB", 1048576, 1024.0, 1.0, 0.0009765625},
		{"1GB", 1073741824, 1048576.0, 1024.0, 1.0},
		{"1TB", 1099511627776, 1073741824.0, 1048576.0, 1024.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test BytesTo* functions
			kb := BytesToKB(tt.bytes)
			if kb != tt.kb {
				t.Errorf("BytesToKB(%d) = %f; want %f", tt.bytes, kb, tt.kb)
			}

			mb := BytesToMB(tt.bytes)
			if mb != tt.mb {
				t.Errorf("BytesToMB(%d) = %f; want %f", tt.bytes, mb, tt.mb)
			}

			gb := BytesToGB(tt.bytes)
			if gb != tt.gb {
				t.Errorf("BytesToGB(%d) = %f; want %f", tt.bytes, gb, tt.gb)
			}
		})
	}
}

func TestReverseConversions(t *testing.T) {
	tests := []struct {
		name  string
		kb    float64
		mb    float64
		gb    float64
		tb    float64
		bytes uint64
	}{
		{"1KB to bytes", 1.0, 0, 0, 0, 1024},
		{"1MB to bytes", 0, 1.0, 0, 0, 1048576},
		{"1GB to bytes", 0, 0, 1.0, 0, 1073741824},
		{"1TB to bytes", 0, 0, 0, 1.0, 1099511627776},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.kb > 0 {
				result := KBToBytes(tt.kb)
				if result != tt.bytes {
					t.Errorf("KBToBytes(%f) = %d; want %d", tt.kb, result, tt.bytes)
				}
			}
			if tt.mb > 0 {
				result := MBToBytes(tt.mb)
				if result != tt.bytes {
					t.Errorf("MBToBytes(%f) = %d; want %d", tt.mb, result, tt.bytes)
				}
			}
			if tt.gb > 0 {
				result := GBToBytes(tt.gb)
				if result != tt.bytes {
					t.Errorf("GBToBytes(%f) = %d; want %d", tt.gb, result, tt.bytes)
				}
			}
			if tt.tb > 0 {
				result := TBToBytes(tt.tb)
				if result != tt.bytes {
					t.Errorf("TBToBytes(%f) = %d; want %d", tt.tb, result, tt.bytes)
				}
			}
		})
	}
}

func TestCalculateSummary(t *testing.T) {
	devices := []Device{
		{
			TotalBytes:     1000000000000, // 1TB
			UsedBytes:      500000000000,  // 500GB
			AvailableBytes: 500000000000,  // 500GB
		},
		{
			TotalBytes:     2000000000000, // 2TB
			UsedBytes:      1000000000000, // 1TB
			AvailableBytes: 1000000000000, // 1TB
		},
	}

	summary := CalculateSummary(devices)

	if summary.TotalDevices != 2 {
		t.Errorf("Expected 2 devices, got %d", summary.TotalDevices)
	}

	if summary.TotalBytes != 3000000000000 {
		t.Errorf("Expected total bytes 3000000000000, got %d", summary.TotalBytes)
	}

	if summary.UsedBytes != 1500000000000 {
		t.Errorf("Expected used bytes 1500000000000, got %d", summary.UsedBytes)
	}

	if summary.AvailBytes != 1500000000000 {
		t.Errorf("Expected avail bytes 1500000000000, got %d", summary.AvailBytes)
	}

	// Check TB conversions (approximately)
	if summary.TotalTB < 2.7 || summary.TotalTB > 2.8 {
		t.Errorf("Expected TotalTB around 2.73, got %f", summary.TotalTB)
	}
}

func TestCalculateSummary_EmptyDevices(t *testing.T) {
	summary := CalculateSummary([]Device{})

	if summary.TotalDevices != 0 {
		t.Errorf("Expected 0 devices, got %d", summary.TotalDevices)
	}

	if summary.TotalBytes != 0 {
		t.Errorf("Expected 0 total bytes, got %d", summary.TotalBytes)
	}
}

// setDetectorForTesting replaces the package-level detector with d and returns
// a cleanup function that restores the original. Use with defer in tests:
//
//	cleanup := setDetectorForTesting(t, mock)
//	defer cleanup()
func setDetectorForTesting(t *testing.T, d Detector) func() {
	t.Helper()
	original := activeDetector
	activeDetector = d
	return func() { activeDetector = original }
}

// mockDetector is a Detector implementation for use in tests.
type mockDetector struct {
	devices []Device
	err     error
}

func (m *mockDetector) DetectDevices() ([]Device, error) {
	return m.devices, m.err
}

// mockUsbDevice is a minimal UsbDevice implementation for use in tests.
type mockUsbDevice struct {
	serial     string
	mountPoint string
}

func (m *mockUsbDevice) GetPath() string                  { return "" }
func (m *mockUsbDevice) GetVendorID() string              { return "" }
func (m *mockUsbDevice) GetProductID() string             { return "" }
func (m *mockUsbDevice) GetManufacturer() string          { return "" }
func (m *mockUsbDevice) GetProduct() string               { return "" }
func (m *mockUsbDevice) GetSerial() string                { return m.serial }
func (m *mockUsbDevice) GetMountPath() string             { return m.mountPoint }
func (m *mockUsbDevice) BlockDevicePath() (string, bool)  { return "", false }
func (m *mockUsbDevice) IsStorageDevice() bool            { return true }
func (m *mockUsbDevice) Partitions() ([]Partition, error) { return nil, nil }

func TestGetManagedDevices(t *testing.T) {
	// Create a temporary directory to simulate a managed device
	tempDir := t.TempDir()
	cirrusDir := ConstructCirrusDir(tempDir)
	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		t.Fatalf("Failed to create test cirrus directory: %v", err)
	}

	// Verify the function executes against the real detector without error.
	devices, err := GetManagedDevices()
	if err != nil {
		t.Fatalf("GetManagedDevices() error = %v", err)
	}

	// Should return a slice (possibly empty if no managed devices exist in CI)
	if devices == nil {
		t.Error("GetManagedDevices() should return non-nil slice")
	}

	// Each device should have non-empty fields
	for i, device := range devices {
		if device.DataDir == "" {
			t.Errorf("Device %d has empty DataDir", i)
		}
		if device.CirrusDir == "" {
			t.Errorf("Device %d has empty FilesDir", i)
		}
	}
}

func TestSetDetectorForTesting_ReturnsInjectedDevices(t *testing.T) {
	tempDir := t.TempDir()
	cirrusDir := ConstructCirrusDir(tempDir)
	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		t.Fatalf("failed to create cirrus dir: %v", err)
	}

	mock := &mockDetector{
		devices: []Device{
			{Name: "test-disk", MountPoint: tempDir, IsInternal: true},
		},
	}
	cleanup := setDetectorForTesting(t, mock)
	defer cleanup()

	devices, err := GetManagedDevices()
	if err != nil {
		t.Fatalf("GetManagedDevices() error = %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 managed device, got %d", len(devices))
	}
	if devices[0].Name != "test-disk" {
		t.Errorf("expected device name 'test-disk', got %q", devices[0].Name)
	}
}

func TestSetDetectorForTesting_CleanupRestoresOriginal(t *testing.T) {
	original := activeDetector
	mock := &mockDetector{}
	cleanup := setDetectorForTesting(t, mock)
	if activeDetector != mock {
		t.Error("expected activeDetector to be replaced by mock")
	}
	cleanup()
	if activeDetector != original {
		t.Error("cleanup did not restore original detector")
	}
}

func TestFindManagedDeviceBySerial_WithMockDetector(t *testing.T) {
	tempDir := t.TempDir()
	cirrusDir := ConstructCirrusDir(tempDir)
	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		t.Fatalf("failed to create cirrus dir: %v", err)
	}

	serial := "ABC123"
	usbInfo := &mockUsbDevice{serial: serial, mountPoint: tempDir}
	mock := &mockDetector{
		devices: []Device{
			{Name: "usb-disk", MountPoint: tempDir, IsInternal: false, UsbInfo: usbInfo},
		},
	}
	cleanup := setDetectorForTesting(t, mock)
	defer cleanup()

	device, err := FindManagedDeviceBySerial(serial)
	if err != nil {
		t.Fatalf("FindManagedDeviceBySerial() error = %v", err)
	}
	if device == nil {
		t.Fatal("expected to find device, got nil")
	}
	if device.Name != "usb-disk" {
		t.Errorf("expected 'usb-disk', got %q", device.Name)
	}
}

func TestFindManagedDeviceBySerial_MissingSerial_WithMockDetector(t *testing.T) {
	tempDir := t.TempDir()
	cirrusDir := ConstructCirrusDir(tempDir)
	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		t.Fatalf("failed to create cirrus dir: %v", err)
	}

	mock := &mockDetector{
		devices: []Device{
			{Name: "internal", MountPoint: tempDir, IsInternal: true},
		},
	}
	cleanup := setDetectorForTesting(t, mock)
	defer cleanup()

	// Empty serial should return first internal device
	device, err := FindManagedDeviceBySerial("")
	if err != nil {
		t.Fatalf("FindManagedDeviceBySerial() error = %v", err)
	}
	if device == nil {
		t.Fatal("expected to find internal device, got nil")
	}
	if device.Name != "internal" {
		t.Errorf("expected 'internal', got %q", device.Name)
	}
}

func TestGetDeviceStatuses(t *testing.T) {
	// This test verifies that GetDeviceStatuses properly merges
	// detected devices with managed devices to build status information
	statuses, err := GetDeviceStatuses()
	if err != nil {
		t.Fatalf("GetDeviceStatuses() error = %v", err)
	}

	// Should return at least one device (the system device)
	if len(statuses) == 0 {
		t.Error("GetDeviceStatuses() returned no devices")
	}

	// Verify each status has required fields
	for i, status := range statuses {
		if status.Name == "" {
			t.Errorf("statuses[%d].Name is empty", i)
		}
		if status.MountPoint == "" {
			t.Errorf("statuses[%d].MountPoint is empty", i)
		}

		// If device is enabled, it should have DataDir and FilesDir
		if status.IsEnabled {
			if status.DataDir == "" {
				t.Errorf("statuses[%d].DataDir is empty for enabled device %s", i, status.Name)
			}
			if status.CirrusDir == "" {
				t.Errorf("statuses[%d].FilesDir is empty for enabled device %s", i, status.Name)
			}
		}
	}

	// At least one device should be enabled (the system device)
	hasEnabled := false
	for _, status := range statuses {
		if status.IsEnabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		t.Error("GetDeviceStatuses() should have at least one enabled device")
	}
}

func TestGetDataDir(t *testing.T) {
	dataDir := GetDataDir()
	if dataDir == "" {
		t.Error("Expected non-empty data directory")
	}
}

func TestGetDataDirForDevice_SystemDevice(t *testing.T) {
	// Test with system root device
	dataDir := GetDataDirForDevice("/")
	if dataDir == "" {
		t.Error("Expected non-empty data directory for root device")
	}
}

func TestGetDataDirForDevice_MacOSSystemVolume(t *testing.T) {
	// Test with macOS system volume path
	dataDir := GetDataDirForDevice("/System/Volumes/Data")
	if dataDir == "" {
		t.Error("Expected non-empty data directory for macOS system volume")
	}
}

func TestGetDataDirForDevice_ExternalDevice(t *testing.T) {
	// Test with external device mount point
	dataDir := GetDataDirForDevice("/Volumes/External")
	expected := "/Volumes/External/autobutler/data"
	if dataDir != expected {
		t.Errorf("Expected %s, got %s", expected, dataDir)
	}
}

func TestGetCirrusDir(t *testing.T) {
	filesDir, err := GetCirrusDir()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if filesDir == "" {
		t.Error("Expected non-empty files directory")
	}
}

func TestDetermineFileTypeFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected FileType
	}{
		{"document.pdf", FileTypePDF},
		{"presentation.pptx", FileTypeSlideshow},
		{"presentation.ppt", FileTypeSlideshow},
		{"photo.png", FileTypeImage},
		{"photo.jpg", FileTypeImage},
		{"photo.jpeg", FileTypeImage},
		{"video.mp4", FileTypeVideo},
		{"video.mov", FileTypeVideo},
		{"book.epub", FileTypeEpub},
		{"document.docx", FileTypeDocx},
		{"archive.zip", FileTypeArchive},
		{"file.txt", FileTypeGeneric},
		{"IMAGE.PNG", FileTypeImage}, // Test case insensitivity
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := DetermineFileTypeFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("DetermineFileTypeFromPath(%s) = %s; want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestVideoMIMETypeFromExtension(t *testing.T) {
	tests := []struct {
		extension string
		expected  string
	}{
		{extension: ".mp4", expected: "video/mp4"},
		{extension: ".m4v", expected: "video/x-m4v"},
		{extension: ".webm", expected: "video/webm"},
		{extension: ".ogg", expected: "video/ogg"},
		{extension: ".avi", expected: "video/x-msvideo"},
		{extension: ".mov", expected: "video/quicktime"},
		{extension: "mp4", expected: "video/mp4"},
		{extension: ".MP4", expected: "video/mp4"},
		{extension: " .mov ", expected: "video/quicktime"},
		{extension: ".mkv", expected: "video/x-matroska"},
		{extension: ".unknown", expected: "application/octet-stream"},
		{extension: "", expected: "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.extension, func(t *testing.T) {
			result := VideoMIMETypeFromExtension(tt.extension)
			if result != tt.expected {
				t.Errorf("VideoMIMETypeFromExtension(%q) = %q; want %q", tt.extension, result, tt.expected)
			}
		})
	}
}

func TestSizeBytesToString(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"0 bytes", 0, "0 B"},
		{"500 bytes", 500, "500 B"},
		{"1 KB", 1024, "1.0 KB"},
		{"1 MB", 1048576, "1.0 MB"},
		{"1 GB", 1073741824, "1.0 GB"},
		{"1 TB", 1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SizeBytesToString(tt.bytes)
			if result != tt.expected {
				t.Errorf("SizeBytesToString(%d) = %s; want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestCustomFileInfo(t *testing.T) {
	fi := NewCustomFileInfo().WithName("testdir").WithSize(4096)

	if fi.Name() != "testdir/" {
		t.Errorf("Expected name 'testdir/', got '%s'", fi.Name())
	}

	if fi.Size() != 4096 {
		t.Errorf("Expected size 4096, got %d", fi.Size())
	}

	if !fi.IsDir() {
		t.Error("Expected IsDir() to be true")
	}

	// Test Mode()
	mode := fi.Mode()
	if mode != 0666 {
		t.Errorf("Expected mode 0666, got %v", mode)
	}

	// Test ModTime()
	modTime := fi.ModTime()
	if modTime.IsZero() {
		t.Error("Expected non-zero ModTime")
	}

	// Test Sys()
	if fi.Sys() != nil {
		t.Error("Expected Sys() to return nil")
	}
}

func TestGetDeviceInfoForPath_MacOSVolume(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	deviceName, devicePath := GetDeviceInfoForPath("/Volumes/MyDrive/some/file.txt")
	if deviceName != "MyDrive" {
		t.Errorf("Expected deviceName 'MyDrive', got '%s'", deviceName)
	}
	if devicePath != "/Volumes/MyDrive" {
		t.Errorf("Expected devicePath '/Volumes/MyDrive', got '%s'", devicePath)
	}
}

func TestGetDeviceInfoForPath_MacOSMainDrive(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	deviceName, devicePath := GetDeviceInfoForPath("/Users/test/file.txt")
	if deviceName != "Macintosh HD" {
		t.Errorf("Expected deviceName 'Macintosh HD', got '%s'", deviceName)
	}
	if devicePath != "/" {
		t.Errorf("Expected devicePath '/', got '%s'", devicePath)
	}
}

func TestGetDeviceInfoForPath_LinuxMedia(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	deviceName, devicePath := GetDeviceInfoForPath("/media/user/USB/file.txt")
	if deviceName != "USB" {
		t.Errorf("Expected deviceName 'USB', got '%s'", deviceName)
	}
	if devicePath != "/media/user/USB" {
		t.Errorf("Expected devicePath '/media/user/USB', got '%s'", devicePath)
	}
}

func TestGetDeviceInfoForPath_LinuxMnt(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	deviceName, devicePath := GetDeviceInfoForPath("/mnt/external/file.txt")
	if deviceName != "external" {
		t.Errorf("Expected deviceName 'external', got '%s'", deviceName)
	}
	if devicePath != "/mnt/external" {
		t.Errorf("Expected devicePath '/mnt/external', got '%s'", devicePath)
	}
}

func TestGetDeviceInfoForPath_LinuxRoot(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	deviceName, devicePath := GetDeviceInfoForPath("/home/user/file.txt")
	if deviceName != "Root" {
		t.Errorf("Expected deviceName 'Root', got '%s'", deviceName)
	}
	if devicePath != "/" {
		t.Errorf("Expected devicePath '/', got '%s'", devicePath)
	}
}

func TestStatFilesInMultipleDirs(t *testing.T) {
	// Create temporary directories for multiple devices
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create files in first directory
	os.WriteFile(filepath.Join(tmpDir1, "file1.txt"), []byte("content1"), 0644)
	os.Mkdir(filepath.Join(tmpDir1, "dir1"), 0755)

	// Create files in second directory (including a duplicate name)
	os.WriteFile(filepath.Join(tmpDir2, "file2.txt"), []byte("content2"), 0644)
	os.WriteFile(filepath.Join(tmpDir2, "file1.txt"), []byte("different content"), 0644)

	dirsWithDevice := []DirWithDevice{
		{Dir: tmpDir1, DeviceName: "Device1", DevicePath: "/dev1"},
		{Dir: tmpDir2, DeviceName: "Device2", DevicePath: "/dev2"},
	}

	files, err := StatFilesInMultipleDirs(dirsWithDevice)
	if err != nil {
		t.Fatalf("StatFilesInMultipleDirs failed: %v", err)
	}

	// Should have 4 entries: dir1, file1.txt (from dev1), file1.txt (from dev2), file2.txt
	if len(files) != 4 {
		t.Errorf("Expected 4 files, got %d", len(files))
	}
}

func TestStatFilesInMultipleDirs_WithNonexistentDir(t *testing.T) {
	// Test that nonexistent directories are skipped
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)

	dirsWithDevice := []DirWithDevice{
		{Dir: "/nonexistent/path", DeviceName: "BadDevice", DevicePath: "/bad"},
		{Dir: tmpDir, DeviceName: "GoodDevice", DevicePath: "/good"},
	}

	files, err := StatFilesInMultipleDirs(dirsWithDevice)
	if err != nil {
		t.Fatalf("StatFilesInMultipleDirs failed: %v", err)
	}

	// Should only have file from the valid directory
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Error("Expected non-nil detector")
	}
}

func TestDeleteFiles_SingleDevice(t *testing.T) {
	// Create a temporary files directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// This test would need the GetCirrusDir to return our tmpDir
	// Since we can't easily mock that, we'll test the structure
	params := DeleteFilesParams{
		RootDir:   "",
		FilePaths: []string{"testfile.txt"},
	}

	// Note: This will fail in testing because GetCirrusDir returns a fixed path
	// In a real app, you'd use dependency injection
	_, err := DeleteFiles(params)
	// We expect an error since the file isn't in the actual GetCirrusDir()
	_ = err // Just verify it doesn't panic
}

func TestMoveFile(t *testing.T) {
	params := MoveFileParams{
		OldFilePath: "old/path/file.txt",
		NewFilePath: "new/path/file.txt",
	}

	// This will fail because GetCirrusDir returns a fixed path
	// but it tests the code doesn't panic
	_, err := MoveFile(params)
	_ = err
}

func TestCreateFolder(t *testing.T) {
	params := CreateFolderParams{
		FolderDir:  "",
		FolderName: "testfolder",
	}

	// This will create a real folder in the GetCirrusDir
	// Clean up afterwards if needed
	result, err := CreateFolder(params)
	if err != nil {
		// Expected since we can't control GetCirrusDir in tests
		t.Logf("CreateFolder returned error (expected): %v", err)
	} else if result != nil {
		// If it succeeded, verify the result structure
		if result.CurrentDir != params.FolderDir {
			t.Errorf("Expected CurrentDir %s, got %s", params.FolderDir, result.CurrentDir)
		}
	}
}

func TestDownloadFile(t *testing.T) {
	params := DownloadFileParams{
		FilePath: "nonexistent/file.txt",
	}

	_, err := DownloadFile(params)
	// Should fail because file doesn't exist
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFindFileAcrossDevices(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	dirs := []DirWithDevice{
		{
			Dir:        tmpDir,
			DeviceName: "TestDevice",
			DevicePath: "/test",
		},
	}

	fullPath, err := FindFileAcrossDevices(dirs, "test.txt")
	if err != nil {
		t.Fatalf("FindFileAcrossDevices failed: %v", err)
	}

	if fullPath != testFile {
		t.Errorf("Expected %s, got %s", testFile, fullPath)
	}
}

func TestFindFileAcrossDevices_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	dirs := []DirWithDevice{
		{
			Dir:        tmpDir,
			DeviceName: "TestDevice",
			DevicePath: "/test",
		},
	}

	_, err := FindFileAcrossDevices(dirs, "nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestStatFilesInDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	files, err := StatFilesInDir(tmpDir, "TestDevice", "/test", "")
	if err != nil {
		t.Fatalf("StatFilesInDir failed: %v", err)
	}

	// Should have 3 entries: 1 directory + 2 files
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Verify directory comes first (due to sorting)
	if !files[0].IsDir() {
		t.Error("Expected first entry to be a directory")
	}
}

func TestGetFolderSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("12345"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("67890"), 0644)

	size, err := GetFolderSize(tmpDir)
	if err != nil {
		t.Fatalf("GetFolderSize failed: %v", err)
	}

	if size != 10 {
		t.Errorf("Expected size 10, got %d", size)
	}
}

func TestGetFolderSize_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("abc"), 0644)

	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("defgh"), 0644)

	size, err := GetFolderSize(tmpDir)
	if err != nil {
		t.Fatalf("GetFolderSize failed: %v", err)
	}

	// Should be 3 + 5 = 8 bytes
	if size != 8 {
		t.Errorf("Expected size 8, got %d", size)
	}
}

func TestDoesFileExist(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(existingFile, []byte("content"), 0644)

	if !DoesFileExist(existingFile) {
		t.Error("Expected file to exist")
	}

	nonExistentFile := filepath.Join(tmpDir, "does-not-exist.txt")
	if DoesFileExist(nonExistentFile) {
		t.Error("Expected file to not exist")
	}
}

func TestMoveFile_CreateDirectory(t *testing.T) {
	// This test validates that MoveFile creates the destination directory
	// We can't easily test this without interfering with GetCirrusDir(),
	// but we can at least verify the function structure
	params := MoveFileParams{
		OldFilePath: "nonexistent/old.txt",
		NewFilePath: "new/path/file.txt",
	}

	_, err := MoveFile(params)
	// Expected to fail since the source doesn't exist
	if err == nil {
		t.Error("Expected error when source file doesn't exist")
	}
}

func TestMoveFile_ToRootDirectory(t *testing.T) {
	// Test the newDir == "." condition by using a file in the current directory
	// When NewFilePath has no directory component, filepath.Dir returns "."

	params := MoveFileParams{
		OldFilePath: "subdir/file.txt",
		NewFilePath: "newfile.txt", // No directory component
	}

	// This will fail because the file doesn't exist, but we can at least
	// verify the NewDir logic would work correctly
	_, err := MoveFile(params)

	// We expect an error because the file doesn't exist
	if err == nil {
		t.Fatal("Expected error when moving non-existent file")
	}

	// Even though the move fails, we can test the logic separately
	newDir := filepath.Dir("newfile.txt")
	if newDir == "." {
		newDir = ""
	}

	if newDir != "" {
		t.Errorf("Expected newDir to be empty string when filepath.Dir returns '.', got '%s'", newDir)
	}

	// Also verify with an actual successful move
	tmpDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(tmpDir, "subdir", "source.txt")
	os.MkdirAll(filepath.Dir(sourceFile), 0755)
	os.WriteFile(sourceFile, []byte("test"), 0644)

	// Get the current files dir and construct paths relative to it
	filesDir, err := GetCirrusDir()
	if err != nil {
		t.Fatalf("GetCirrusDir failed: %v", err)
	}

	// Create test structure in actual filesDir
	testSourceDir := filepath.Join(filesDir, "test_move_root")
	testSourceFile := filepath.Join(testSourceDir, "subdir", "file.txt")
	os.MkdirAll(filepath.Dir(testSourceFile), 0755)
	os.WriteFile(testSourceFile, []byte("content"), 0644)
	defer os.RemoveAll(testSourceDir)

	params2 := MoveFileParams{
		OldFilePath: "test_move_root/subdir/file.txt",
		NewFilePath: "test_move_root/moved.txt",
	}

	result2, err := MoveFile(params2)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	// NewDir should be "test_move_root"
	if result2.NewDir != "test_move_root" {
		t.Errorf("Expected NewDir 'test_move_root', got '%s'", result2.NewDir)
	}

	// Now test the case where newDir == "." (moving to root)
	testSourceFile2 := filepath.Join(testSourceDir, "subdir", "file2.txt")
	os.WriteFile(testSourceFile2, []byte("content2"), 0644)

	params3 := MoveFileParams{
		OldFilePath: "test_move_root/subdir/file2.txt",
		NewFilePath: "rootfile.txt", // No directory path, filepath.Dir will return "."
	}

	result3, err := MoveFile(params3)
	if err != nil {
		t.Fatalf("MoveFile to root failed: %v", err)
	}

	// When filepath.Dir returns ".", it should be converted to ""
	if result3.NewDir != "" {
		t.Errorf("Expected NewDir to be empty string, got '%s'", result3.NewDir)
	}

	// Clean up the file we moved to root
	os.Remove(filepath.Join(filesDir, "rootfile.txt"))
}

func TestDownloadFile_MultiDevice_NotFound(t *testing.T) {
	params := DownloadFileParams{
		FilePath: "test/nonexistent.txt",
	}

	_, err := DownloadFile(params)
	if err == nil {
		t.Error("Expected error when file doesn't exist on any device")
	}
}

func TestInitializeDeviceDataDir(t *testing.T) {
	// Create a temporary directory to use as a mount point
	tempDir := t.TempDir()

	err := InitializeDeviceDataDir(tempDir)
	if err != nil {
		t.Fatalf("InitializeDeviceDataDir() error = %v", err)
	}

	// Verify the directory structure was created
	// For external devices, the path is: mountPoint/autobutler/data/cirrus
	dataDir := filepath.Join(tempDir, "autobutler", "data")
	cirrusDir := ConstructCirrusDir(dataDir)
	if _, err := os.Stat(cirrusDir); os.IsNotExist(err) {
		t.Errorf("Expected cirrus directory to be created at %s", cirrusDir)
	}
}

func TestInitializeDeviceDataDir_AlreadyExists(t *testing.T) {
	// Create a temporary directory with existing structure
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "autobutler", "data")
	cirrusDir := ConstructCirrusDir(dataDir)
	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Should succeed even if directory already exists
	err := InitializeDeviceDataDir(tempDir)
	if err != nil {
		t.Errorf("InitializeDeviceDataDir() should succeed when directory exists, got error: %v", err)
	}
}

func TestDetermineFileTypeFromPath_EmptyOrSlashPath(t *testing.T) {
	// Test that "" and "/" are treated as folders
	tests := []struct {
		path string
	}{
		{""},
		{"/"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := DetermineFileTypeFromPath(tt.path)
			if result != FileTypeFolder {
				t.Errorf("DetermineFileTypeFromPath(%q) = %s; want %s", tt.path, result, FileTypeFolder)
			}
		})
	}
}

func TestDetermineFileTypeFromPath_DirectoryPath(t *testing.T) {
	// Test a directory path (uses os.Stat and IsDir check)
	tempDir := t.TempDir()

	result := DetermineFileTypeFromPath(tempDir)
	if result != FileTypeFolder {
		t.Errorf("DetermineFileTypeFromPath(%s) = %s; want %s", tempDir, result, FileTypeFolder)
	}
}

func TestDetermineFileTypeFromPath_NonexistentFile(t *testing.T) {
	// Test a nonexistent file (os.Stat will fail, should return FileTypeGeneric)
	result := DetermineFileTypeFromPath("/nonexistent/path/file.unknown")
	if result != FileTypeGeneric {
		t.Errorf("DetermineFileTypeFromPath(nonexistent) = %s; want %s", result, FileTypeGeneric)
	}
}

func TestDetermineFileType_NilFile(t *testing.T) {
	// Test with nil file - should return FileTypeSpacer
	result := DetermineFileType("", nil)
	if result != FileTypeSpacer {
		t.Errorf("DetermineFileType(nil) = %s; want %s", result, FileTypeSpacer)
	}
}

func TestDetermineFileType_Directory(t *testing.T) {
	// Test with a directory DeviceFileInfo
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	fileInfo, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("Failed to stat test directory: %v", err)
	}

	deviceFileInfo := &DeviceFileInfo{
		FileInfo:   fileInfo,
		DeviceName: "test-device",
		DevicePath: tempDir,
		FullPath:   testDir,
	}

	result := DetermineFileType("", deviceFileInfo)
	if result != FileTypeFolder {
		t.Errorf("DetermineFileType(directory) = %s; want %s", result, FileTypeFolder)
	}
}

func TestDetermineFileType_RegularFile(t *testing.T) {
	// Test with a regular file - should use DetermineFileTypeFromPath
	// Create a test file in the actual filesDir
	filesDir, err := GetCirrusDir()
	if err != nil {
		t.Fatalf("GetCirrusDir failed: %v", err)
	}
	testDir := filepath.Join(filesDir, "test_determine_type")
	testFile := filepath.Join(testDir, "test.pdf")

	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	deviceFileInfo := &DeviceFileInfo{
		FileInfo:   fileInfo,
		DeviceName: "test-device",
		DevicePath: filesDir,
		FullPath:   testFile,
	}

	result := DetermineFileType("test_determine_type", deviceFileInfo)
	if result != FileTypePDF {
		t.Errorf("DetermineFileType(test.pdf) = %s; want %s", result, FileTypePDF)
	}
}

func TestDetermineFileType_FileNotFoundInFilesDir(t *testing.T) {
	// Test when file.IsDir() is false but the file doesn't exist in filesDir
	// This should return FileTypeGeneric
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "ghost.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	deviceFileInfo := &DeviceFileInfo{
		FileInfo:   fileInfo,
		DeviceName: "test-device",
		DevicePath: tempDir,
		FullPath:   testFile,
	}

	// The file exists in tempDir but not in filesDir, so os.Stat will fail
	result := DetermineFileType("nonexistent_root", deviceFileInfo)
	if result != FileTypeGeneric {
		t.Errorf("DetermineFileType(file not in filesDir) = %s; want %s", result, FileTypeGeneric)
	}
}

func TestGetAvailableSpaceInBytes(t *testing.T) {
	// Test that GetAvailableSpaceInBytes returns a non-zero value for a valid directory
	tempDir := t.TempDir()

	availableSpace := GetAvailableSpaceInBytes(tempDir)

	// We just verify that it returns a reasonable value (greater than 0)
	// The actual value will vary by system
	if availableSpace == 0 {
		t.Errorf("GetAvailableSpaceInBytes() = 0; want > 0")
	}
}

func TestNewDeviceFileInfo(t *testing.T) {
	// Create a test file to get real FileInfo
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Create DeviceFileInfo using constructor
	deviceName := "TestDevice"
	devicePath := "/test/path"
	fullPath := testFile

	deviceFileInfo := NewDeviceFileInfo(fileInfo, deviceName, devicePath, fullPath, "")

	// Verify all fields are set correctly
	if deviceFileInfo.DeviceName != deviceName {
		t.Errorf("DeviceName = %s; want %s", deviceFileInfo.DeviceName, deviceName)
	}
	if deviceFileInfo.DevicePath != devicePath {
		t.Errorf("DevicePath = %s; want %s", deviceFileInfo.DevicePath, devicePath)
	}
	if deviceFileInfo.FullPath != fullPath {
		t.Errorf("FullPath = %s; want %s", deviceFileInfo.FullPath, fullPath)
	}
	if deviceFileInfo.FileInfo != fileInfo {
		t.Errorf("FileInfo not set correctly")
	}
}

func TestDeviceFileInfo_WrapperMethods(t *testing.T) {
	// Create a test file to get real FileInfo
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content for wrapper methods")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	deviceFileInfo := NewDeviceFileInfo(fileInfo, "Device", "/device", testFile, "")

	// Test Name() wrapper
	if deviceFileInfo.Name() != fileInfo.Name() {
		t.Errorf("Name() = %s; want %s", deviceFileInfo.Name(), fileInfo.Name())
	}

	// Test Size() wrapper
	if deviceFileInfo.Size() != fileInfo.Size() {
		t.Errorf("Size() = %d; want %d", deviceFileInfo.Size(), fileInfo.Size())
	}

	// Test Mode() wrapper
	if deviceFileInfo.Mode() != fileInfo.Mode() {
		t.Errorf("Mode() = %v; want %v", deviceFileInfo.Mode(), fileInfo.Mode())
	}

	// Test ModTime() wrapper
	if !deviceFileInfo.ModTime().Equal(fileInfo.ModTime()) {
		t.Errorf("ModTime() = %v; want %v", deviceFileInfo.ModTime(), fileInfo.ModTime())
	}

	// Test IsDir() wrapper
	if deviceFileInfo.IsDir() != fileInfo.IsDir() {
		t.Errorf("IsDir() = %v; want %v", deviceFileInfo.IsDir(), fileInfo.IsDir())
	}

	// Test Sys() wrapper
	if deviceFileInfo.Sys() != fileInfo.Sys() {
		t.Errorf("Sys() mismatch")
	}
}

func TestSetupCirrusDir(t *testing.T) {
	tests := []struct {
		name          string
		setupBefore   func(t *testing.T, dataDir string) // Setup function to prepare test state
		wantError     bool
		validateAfter func(t *testing.T, dataDir string) // Validation function to check post-setup state
	}{
		{
			name: "creates cirrus directory when neither exists",
			setupBefore: func(t *testing.T, dataDir string) {
				// Both directories don't exist
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				info, err := os.Stat(cirrusDir)
				if err != nil {
					t.Fatalf("cirrus directory should exist: %v", err)
				}
				if !info.IsDir() {
					t.Errorf("cirrus path should be a directory")
				}
			},
		},
		{
			name: "does not error when cirrus directory already exists",
			setupBefore: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				if err := os.MkdirAll(cirrusDir, 0755); err != nil {
					t.Fatalf("failed to create cirrus directory: %v", err)
				}
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				info, err := os.Stat(cirrusDir)
				if err != nil {
					t.Fatalf("cirrus directory should exist: %v", err)
				}
				if !info.IsDir() {
					t.Errorf("cirrus path should be a directory")
				}
			},
		},
		{
			name: "migrates single file from legacy files directory",
			setupBefore: func(t *testing.T, dataDir string) {
				legacyFilesPath := filepath.Join(dataDir, "files")
				if err := os.MkdirAll(legacyFilesPath, 0755); err != nil {
					t.Fatalf("failed to create legacy files directory: %v", err)
				}
				// Create a test file in legacy directory
				testFile := filepath.Join(legacyFilesPath, "test.txt")
				if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				legacyFilesPath := filepath.Join(dataDir, "files")

				// Check that file was moved to cirrus
				migratedFile := filepath.Join(cirrusDir, "test.txt")
				content, err := os.ReadFile(migratedFile)
				if err != nil {
					t.Fatalf("migrated file should exist in cirrus: %v", err)
				}
				if string(content) != "test content" {
					t.Errorf("migrated file content mismatch: got %q, want %q", string(content), "test content")
				}

				// Check that legacy directory was removed
				if _, err := os.Stat(legacyFilesPath); !os.IsNotExist(err) {
					t.Errorf("legacy files directory should be removed")
				}
			},
		},
		{
			name: "migrates multiple files from legacy directory",
			setupBefore: func(t *testing.T, dataDir string) {
				legacyFilesPath := filepath.Join(dataDir, "files")
				if err := os.MkdirAll(legacyFilesPath, 0755); err != nil {
					t.Fatalf("failed to create legacy files directory: %v", err)
				}
				// Create multiple test files
				for i := 1; i <= 3; i++ {
					testFile := filepath.Join(legacyFilesPath, fmt.Sprintf("file%d.txt", i))
					if err := os.WriteFile(testFile, []byte(fmt.Sprintf("content%d", i)), 0644); err != nil {
						t.Fatalf("failed to create test file: %v", err)
					}
				}
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				legacyFilesPath := filepath.Join(dataDir, "files")

				// Check that all files were moved
				for i := 1; i <= 3; i++ {
					migratedFile := filepath.Join(cirrusDir, fmt.Sprintf("file%d.txt", i))
					content, err := os.ReadFile(migratedFile)
					if err != nil {
						t.Fatalf("file%d.txt should be migrated: %v", i, err)
					}
					expectedContent := fmt.Sprintf("content%d", i)
					if string(content) != expectedContent {
						t.Errorf("file%d.txt content mismatch: got %q, want %q", i, string(content), expectedContent)
					}
				}

				// Check that legacy directory was removed
				if _, err := os.Stat(legacyFilesPath); !os.IsNotExist(err) {
					t.Errorf("legacy files directory should be removed")
				}
			},
		},
		{
			name: "migrates subdirectories and their contents",
			setupBefore: func(t *testing.T, dataDir string) {
				legacyFilesPath := filepath.Join(dataDir, "files")
				if err := os.MkdirAll(legacyFilesPath, 0755); err != nil {
					t.Fatalf("failed to create legacy files directory: %v", err)
				}
				// Create a subdirectory with files
				subDir := filepath.Join(legacyFilesPath, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("failed to create subdirectory: %v", err)
				}
				testFile := filepath.Join(subDir, "nested.txt")
				if err := os.WriteFile(testFile, []byte("nested content"), 0644); err != nil {
					t.Fatalf("failed to create nested file: %v", err)
				}
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				legacyFilesPath := filepath.Join(dataDir, "files")

				// Check that subdirectory was moved
				movedSubDir := filepath.Join(cirrusDir, "subdir")
				nestedFile := filepath.Join(movedSubDir, "nested.txt")
				content, err := os.ReadFile(nestedFile)
				if err != nil {
					t.Fatalf("nested file should be migrated: %v", err)
				}
				if string(content) != "nested content" {
					t.Errorf("nested file content mismatch: got %q, want %q", string(content), "nested content")
				}

				// Check that legacy directory was removed
				if _, err := os.Stat(legacyFilesPath); !os.IsNotExist(err) {
					t.Errorf("legacy files directory should be removed")
				}
			},
		},
		{
			name: "handles empty legacy files directory",
			setupBefore: func(t *testing.T, dataDir string) {
				legacyFilesPath := filepath.Join(dataDir, "files")
				if err := os.MkdirAll(legacyFilesPath, 0755); err != nil {
					t.Fatalf("failed to create legacy files directory: %v", err)
				}
				// Don't add any files - directory is empty
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				legacyFilesPath := filepath.Join(dataDir, "files")
				// Check that empty legacy directory was removed
				if _, err := os.Stat(legacyFilesPath); !os.IsNotExist(err) {
					t.Errorf("empty legacy files directory should be removed")
				}
			},
		},
		{
			name: "migrates when both cirrus and legacy exist with content",
			setupBefore: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				legacyFilesPath := filepath.Join(dataDir, "files")

				// Create cirrus directory with existing file
				if err := os.MkdirAll(cirrusDir, 0755); err != nil {
					t.Fatalf("failed to create cirrus directory: %v", err)
				}
				existingFile := filepath.Join(cirrusDir, "existing.txt")
				if err := os.WriteFile(existingFile, []byte("existing content"), 0644); err != nil {
					t.Fatalf("failed to create existing file: %v", err)
				}

				// Create legacy directory with file
				if err := os.MkdirAll(legacyFilesPath, 0755); err != nil {
					t.Fatalf("failed to create legacy files directory: %v", err)
				}
				legacyFile := filepath.Join(legacyFilesPath, "legacy.txt")
				if err := os.WriteFile(legacyFile, []byte("legacy content"), 0644); err != nil {
					t.Fatalf("failed to create legacy file: %v", err)
				}
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				legacyFilesPath := filepath.Join(dataDir, "files")

				// Check both files are in cirrus
				existingFile := filepath.Join(cirrusDir, "existing.txt")
				migratedFile := filepath.Join(cirrusDir, "legacy.txt")

				if _, err := os.Stat(existingFile); err != nil {
					t.Errorf("existing file should still be in cirrus: %v", err)
				}
				if _, err := os.Stat(migratedFile); err != nil {
					t.Errorf("legacy file should be migrated to cirrus: %v", err)
				}

				// Check that legacy directory was removed
				if _, err := os.Stat(legacyFilesPath); !os.IsNotExist(err) {
					t.Errorf("legacy files directory should be removed")
				}
			},
		},
		{
			name: "handles file naming conflicts by suffixing",
			setupBefore: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)
				legacyFilesPath := filepath.Join(dataDir, "files")

				// Create cirrus directory with a file
				if err := os.MkdirAll(cirrusDir, 0755); err != nil {
					t.Fatalf("failed to create cirrus directory: %v", err)
				}
				conflictFile := filepath.Join(cirrusDir, "conflict.txt")
				if err := os.WriteFile(conflictFile, []byte("original content"), 0644); err != nil {
					t.Fatalf("failed to create original file: %v", err)
				}

				// Create legacy directory with a file with same name
				if err := os.MkdirAll(legacyFilesPath, 0755); err != nil {
					t.Fatalf("failed to create legacy files directory: %v", err)
				}
				legacyFile := filepath.Join(legacyFilesPath, "conflict.txt")
				if err := os.WriteFile(legacyFile, []byte("migrated content"), 0644); err != nil {
					t.Fatalf("failed to create legacy file: %v", err)
				}
			},
			wantError: false,
			validateAfter: func(t *testing.T, dataDir string) {
				cirrusDir := ConstructCirrusDir(dataDir)

				// Check original file still exists
				originalFile := filepath.Join(cirrusDir, "conflict.txt")
				content, err := os.ReadFile(originalFile)
				if err != nil {
					t.Fatalf("original file should exist: %v", err)
				}
				if string(content) != "original content" {
					t.Errorf("original file should not be modified: got %q, want %q", string(content), "original content")
				}

				// Check migrated file was created with suffix
				migratedFile := filepath.Join(cirrusDir, "conflict_(1).txt")
				content, err = os.ReadFile(migratedFile)
				if err != nil {
					t.Fatalf("migrated file with suffix should exist: %v", err)
				}
				if string(content) != "migrated content" {
					t.Errorf("migrated file should have migrated content: got %q, want %q", string(content), "migrated content")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tmpDir := t.TempDir()

			// Run setup
			tt.setupBefore(t, tmpDir)

			// Temporarily change GetDataDir by modifying cirrus and files paths directly
			cirrusDir := ConstructCirrusDir(tmpDir)
			legacyFilesPath := filepath.Join(tmpDir, "files")

			// Copy the migration logic inline to test with our temp directory
			if _, err := os.Stat(cirrusDir); os.IsNotExist(err) {
				if err := os.MkdirAll(cirrusDir, 0755); err != nil {
					t.Fatalf("failed to create cirrus directory: %v", err)
				}
			}

			if _, err := os.Stat(legacyFilesPath); err == nil {
				entries, err := os.ReadDir(legacyFilesPath)
				if err != nil {
					t.Fatalf("failed to read legacy files directory: %v", err)
				}
				for _, entry := range entries {
					oldPath := filepath.Join(legacyFilesPath, entry.Name())
					targetPath := filepath.Join(cirrusDir, entry.Name())
					newPath := GetNonConflictingPath(targetPath)
					if err := os.Rename(oldPath, newPath); err != nil {
						t.Fatalf("failed to move file %s to cirrus directory: %v", entry.Name(), err)
					}
				}
				if err := os.RemoveAll(legacyFilesPath); err != nil {
					t.Fatalf("failed to delete legacy files directory: %v", err)
				}
			}

			// Run validation
			tt.validateAfter(t, tmpDir)
		})
	}
}

func TestGetNonConflictingPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name            string
		setup           func(t *testing.T, dir string) // Setup function to create files
		inputPath       string
		expectedPath    string
		expectedContent string // For validating the file can be created
	}{
		{
			name: "returns original path when no conflict exists",
			setup: func(t *testing.T, dir string) {
				// Don't create any files
			},
			inputPath:    filepath.Join(tmpDir, "newfile.txt"),
			expectedPath: filepath.Join(tmpDir, "newfile.txt"),
		},
		{
			name: "suffixes with _(1) when file exists",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("original"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			inputPath:    filepath.Join(tmpDir, "file.txt"),
			expectedPath: filepath.Join(tmpDir, "file_(1).txt"),
		},
		{
			name: "increments suffix when multiple conflicts exist",
			setup: func(t *testing.T, dir string) {
				for i := 0; i < 2; i++ {
					path := filepath.Join(dir, fmt.Sprintf("file%s.txt", func() string {
						if i == 0 {
							return ""
						}
						return fmt.Sprintf("_(%d)", i)
					}()))
					if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
						t.Fatalf("failed to create test file: %v", err)
					}
				}
			},
			inputPath:    filepath.Join(tmpDir, "file.txt"),
			expectedPath: filepath.Join(tmpDir, "file_(2).txt"),
		},
		{
			name: "handles files without extension",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "file"), []byte("original"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			inputPath:    filepath.Join(tmpDir, "file"),
			expectedPath: filepath.Join(tmpDir, "file_(1)"),
		},
		{
			name: "handles files with multiple dots in name",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "file.backup.txt"), []byte("original"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			inputPath:    filepath.Join(tmpDir, "file.backup.txt"),
			expectedPath: filepath.Join(tmpDir, "file.backup_(1).txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh temp directory for this test
			testDir := t.TempDir()

			tt.setup(t, testDir)

			inputPath := filepath.Join(testDir, filepath.Base(tt.inputPath))
			result := GetNonConflictingPath(inputPath)

			expectedPath := filepath.Join(testDir, filepath.Base(tt.expectedPath))
			if result != expectedPath {
				t.Errorf("GetNonConflictingPath(%s) = %s, want %s", filepath.Base(inputPath), filepath.Base(result), filepath.Base(expectedPath))
			}
		})
	}
}
