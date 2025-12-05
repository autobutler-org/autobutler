package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

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
}

func TestDeviceApplySimpleCategorization_SystemVolume(t *testing.T) {
	device := &Device{
		MountPoint: "/",
		UsedBytes:  1000000,
	}

	device.ApplySimpleCategorization()

	if device.Categories == nil {
		t.Fatal("Expected Categories to be initialized")
	}

	if device.Categories["system"] != 100000 { // 10%
		t.Errorf("Expected system to be 100000, got %d", device.Categories["system"])
	}
}

func TestDeviceApplySimpleCategorization_ExternalVolume(t *testing.T) {
	device := &Device{
		MountPoint: "/Volumes/External",
		UsedBytes:  5000000,
	}

	device.ApplySimpleCategorization()

	if device.Categories["other"] != 5000000 {
		t.Errorf("Expected all bytes to be categorized as 'other', got %d", device.Categories["other"])
	}
}

func TestGetDataDir(t *testing.T) {
	dataDir := GetDataDir()
	if dataDir == "" {
		t.Error("Expected non-empty data directory")
	}
}

func TestGetFilesDir(t *testing.T) {
	filesDir := GetFilesDir()
	if filesDir == "" {
		t.Error("Expected non-empty files directory")
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

	// This test would need the GetFilesDir to return our tmpDir
	// Since we can't easily mock that, we'll test the structure
	params := DeleteFilesParams{
		RootDir:        "",
		FilePaths:      []string{"testfile.txt"},
		ManagedDevices: nil,
	}

	// Note: This will fail in testing because GetFilesDir returns a fixed path
	// In a real app, you'd use dependency injection
	_, err := DeleteFiles(params)
	// We expect an error since the file isn't in the actual GetFilesDir()
	_ = err // Just verify it doesn't panic
}

func TestMoveFile(t *testing.T) {
	params := MoveFileParams{
		FilePath:    "old/path/file.txt",
		NewFilePath: "new/path/file.txt",
	}

	// This will fail because GetFilesDir returns a fixed path
	// but it tests the code doesn't panic
	_, err := MoveFile(params)
	_ = err
}

func TestCreateFolder(t *testing.T) {
	params := CreateFolderParams{
		FolderDir:  "",
		FolderName: "testfolder",
	}

	// This will create a real folder in the GetFilesDir
	// Clean up afterwards if needed
	result, err := CreateFolder(params)
	if err != nil {
		// Expected since we can't control GetFilesDir in tests
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
		FilePath:       "nonexistent/file.txt",
		ManagedDevices: nil,
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

	files, err := StatFilesInDir(tmpDir, "TestDevice", "/test")
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

func TestCalculateSummary(t *testing.T) {
	devices := []Device{
		{
			TotalBytes: 1000000000000, // 1TB
			UsedBytes:  500000000000,  // 500GB
			AvailBytes: 500000000000,  // 500GB
		},
		{
			TotalBytes: 2000000000000, // 2TB
			UsedBytes:  1000000000000, // 1TB
			AvailBytes: 1000000000000, // 1TB
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

func TestDeleteFiles_MultiDevice(t *testing.T) {
	// Create test directories for multiple devices
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create test files on both devices
	testFile1 := filepath.Join(tmpDir1, "test", "file1.txt")
	testFile2 := filepath.Join(tmpDir2, "test", "file1.txt")
	
	os.MkdirAll(filepath.Dir(testFile1), 0755)
	os.MkdirAll(filepath.Dir(testFile2), 0755)
	os.WriteFile(testFile1, []byte("content1"), 0644)
	os.WriteFile(testFile2, []byte("content2"), 0644)

	// Setup managed devices
	devices := []ManagedDevice{
		{
			Device: Device{
				Name:       "Device1",
				MountPoint: "/mnt/dev1",
			},
			FilesDir: tmpDir1,
		},
		{
			Device: Device{
				Name:       "Device2",
				MountPoint: "/mnt/dev2",
			},
			FilesDir: tmpDir2,
		},
	}

	params := DeleteFilesParams{
		RootDir:        "test",
		FilePaths:      []string{"file1.txt"},
		ManagedDevices: devices,
	}

	result, err := DeleteFiles(params)
	if err != nil {
		t.Fatalf("DeleteFiles failed: %v", err)
	}

	if result.RootDir != "test" {
		t.Errorf("Expected RootDir 'test', got '%s'", result.RootDir)
	}

	// Verify files were deleted from both devices
	if DoesFileExist(testFile1) {
		t.Error("Expected file to be deleted from device 1")
	}
	if DoesFileExist(testFile2) {
		t.Error("Expected file to be deleted from device 2")
	}
}

func TestDeleteFiles_MultiDevice_PartialExistence(t *testing.T) {
	// Create test directories
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create test file only on first device
	testFile1 := filepath.Join(tmpDir1, "test", "file1.txt")
	os.MkdirAll(filepath.Dir(testFile1), 0755)
	os.WriteFile(testFile1, []byte("content1"), 0644)

	devices := []ManagedDevice{
		{
			Device: Device{
				Name:       "Device1",
				MountPoint: "/mnt/dev1",
			},
			FilesDir: tmpDir1,
		},
		{
			Device: Device{
				Name:       "Device2",
				MountPoint: "/mnt/dev2",
			},
			FilesDir: tmpDir2,
		},
	}

	params := DeleteFilesParams{
		RootDir:        "test",
		FilePaths:      []string{"file1.txt"},
		ManagedDevices: devices,
	}

	_, err := DeleteFiles(params)
	if err != nil {
		t.Fatalf("DeleteFiles should not fail when file doesn't exist on all devices: %v", err)
	}

	// Verify file was deleted from device 1
	if DoesFileExist(testFile1) {
		t.Error("Expected file to be deleted from device 1")
	}
}

func TestMoveFile_CreateDirectory(t *testing.T) {
	// This test validates that MoveFile creates the destination directory
	// We can't easily test this without interfering with GetFilesDir(),
	// but we can at least verify the function structure
	params := MoveFileParams{
		FilePath:    "nonexistent/old.txt",
		NewFilePath: "new/path/file.txt",
	}

	_, err := MoveFile(params)
	// Expected to fail since the source doesn't exist
	if err == nil {
		t.Error("Expected error when source file doesn't exist")
	}
}

func TestUploadFiles_FileConflict(t *testing.T) {
	// Test that UploadFiles handles filename conflicts by appending numbers
	// This would require mocking multipart.FileHeader which is complex
	// Skip for now as it requires significant setup
	t.Skip("Requires complex multipart.FileHeader mocking")
}

func TestDownloadFile_MultiDevice(t *testing.T) {
	// Create test directories
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create test file only on second device
	testFile := filepath.Join(tmpDir2, "test", "file1.txt")
	os.MkdirAll(filepath.Dir(testFile), 0755)
	os.WriteFile(testFile, []byte("content"), 0644)

	devices := []ManagedDevice{
		{
			Device: Device{
				Name:       "Device1",
				MountPoint: "/mnt/dev1",
			},
			FilesDir: tmpDir1,
		},
		{
			Device: Device{
				Name:       "Device2",
				MountPoint: "/mnt/dev2",
			},
			FilesDir: tmpDir2,
		},
	}

	params := DownloadFileParams{
		FilePath:       "test/file1.txt",
		ManagedDevices: devices,
	}

	result, err := DownloadFile(params)
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}

	if result.FullPath != testFile {
		t.Errorf("Expected FullPath %s, got %s", testFile, result.FullPath)
	}

	if result.FileType == FileTypeFolder {
		t.Error("Expected FileType to not be FileTypeFolder for a file")
	}
}

func TestDownloadFile_MultiDevice_NotFound(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	devices := []ManagedDevice{
		{
			Device: Device{
				Name:       "Device1",
				MountPoint: "/mnt/dev1",
			},
			FilesDir: tmpDir1,
		},
		{
			Device: Device{
				Name:       "Device2",
				MountPoint: "/mnt/dev2",
			},
			FilesDir: tmpDir2,
		},
	}

	params := DownloadFileParams{
		FilePath:       "test/nonexistent.txt",
		ManagedDevices: devices,
	}

	_, err := DownloadFile(params)
	if err == nil {
		t.Error("Expected error when file doesn't exist on any device")
	}
}

