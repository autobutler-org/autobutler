package fileutil

import (
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
