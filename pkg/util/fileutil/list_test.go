package fileutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// makeManagedDevice creates a ManagedDevice backed by a real temp directory,
// matching the pattern used in storageutil tests.
func makeManagedDevice(t *testing.T, name string) storageutil.ManagedDevice {
	t.Helper()
	dir := t.TempDir()
	filesDir := filepath.Join(dir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}
	return storageutil.ManagedDevice{
		Device: storageutil.Device{
			Name:       name,
			MountPoint: dir,
			IsInternal: true,
		},
		DataDir:  dir,
		FilesDir: filesDir,
	}
}

func TestListFilesImpl_EmptyDevice(t *testing.T) {
	device := makeManagedDevice(t, "test-device")
	result, err := listFilesOnDevices("", []storageutil.ManagedDevice{device})
	if err != nil {
		t.Fatalf("listFilesOnDevices failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 files in empty device, got %d", len(result))
	}
}

func TestListFilesImpl_WithFiles(t *testing.T) {
	device := makeManagedDevice(t, "test-device")

	// Create some files in the files dir
	if err := os.WriteFile(filepath.Join(device.FilesDir, "file1.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(device.FilesDir, "file2.pdf"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := listFilesOnDevices("", []storageutil.ManagedDevice{device})
	if err != nil {
		t.Fatalf("listFilesOnDevices failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 files, got %d", len(result))
	}

	names := make(map[string]bool)
	for _, f := range result {
		names[f.Name] = true
		if f.DeviceName != "test-device" {
			t.Errorf("Expected DeviceName 'test-device', got %q", f.DeviceName)
		}
		if f.IsDir {
			t.Errorf("Expected file, got directory for %q", f.Name)
		}
	}
	if !names["file1.txt"] {
		t.Error("Expected file1.txt in results")
	}
	if !names["file2.pdf"] {
		t.Error("Expected file2.pdf in results")
	}
}

func TestListFilesImpl_WithSubdirectory(t *testing.T) {
	device := makeManagedDevice(t, "test-device")

	subdir := filepath.Join(device.FilesDir, "docs")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := listFilesOnDevices("", []storageutil.ManagedDevice{device})
	if err != nil {
		t.Fatalf("listFilesOnDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 entry (the docs dir), got %d", len(result))
	}
	if result[0].Name != "docs/" || !result[0].IsDir {
		t.Errorf("Expected 'docs/' directory, got %+v", result[0])
	}

	// Now list inside the subdir
	result, err = listFilesOnDevices("docs", []storageutil.ManagedDevice{device})
	if err != nil {
		t.Fatalf("listFilesOnDevices for subdir failed: %v", err)
	}
	if len(result) != 1 || result[0].Name != "readme.txt" {
		t.Errorf("Expected readme.txt in docs/, got %+v", result)
	}
}

func TestListFilesImpl_DeduplicateFolders(t *testing.T) {
	// Two devices both have a "photos" folder — should appear once
	device1 := makeManagedDevice(t, "device1")
	device2 := makeManagedDevice(t, "device2")

	for _, d := range []storageutil.ManagedDevice{device1, device2} {
		if err := os.Mkdir(filepath.Join(d.FilesDir, "photos"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	result, err := listFilesOnDevices("", []storageutil.ManagedDevice{device1, device2})
	if err != nil {
		t.Fatalf("listFilesOnDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 deduplicated 'photos' folder, got %d results", len(result))
	}
	if result[0].Name != "photos/" || !result[0].IsDir {
		t.Errorf("Expected photos/ dir, got %+v", result[0])
	}
}

func TestListFilesImpl_FilesNotDeduplicatedAcrossDevices(t *testing.T) {
	// Files with the same name on two devices should both appear (only folders deduplicate)
	device1 := makeManagedDevice(t, "device1")
	device2 := makeManagedDevice(t, "device2")

	for _, d := range []storageutil.ManagedDevice{device1, device2} {
		if err := os.WriteFile(filepath.Join(d.FilesDir, "backup.zip"), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := listFilesOnDevices("", []storageutil.ManagedDevice{device1, device2})
	if err != nil {
		t.Fatalf("listFilesOnDevices failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 entries (same filename on two devices), got %d", len(result))
	}
}

func TestListFilesImpl_NoDevices(t *testing.T) {
	result, err := listFilesOnDevices("", []storageutil.ManagedDevice{})
	if err != nil {
		t.Fatalf("listFilesOnDevices failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result for no devices, got %d", len(result))
	}
}

func TestListFilesImpl_NonExistentSubdir(t *testing.T) {
	device := makeManagedDevice(t, "test-device")

	// Listing a subdir that doesn't exist should fail so the client can render
	// an explicit invalid-folder state instead of an empty listing.
	result, err := listFilesOnDevices("nonexistent", []storageutil.ManagedDevice{device})
	if err == nil {
		t.Fatal("expected an error for a nonexistent subdir")
	}
	if result != nil {
		t.Errorf("expected no results for nonexistent subdir, got %d", len(result))
	}
}

func TestFileNodeJSON_Fields(t *testing.T) {
	device := makeManagedDevice(t, "my-device")
	if err := os.WriteFile(filepath.Join(device.FilesDir, "test.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := listFilesOnDevices("", []storageutil.ManagedDevice{device})
	if err != nil {
		t.Fatalf("listFilesOnDevices failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(result))
	}

	f := result[0]
	if f.Name != "test.txt" {
		t.Errorf("Expected Name 'test.txt', got %q", f.Name)
	}
	if f.IsDir {
		t.Error("Expected IsDir false")
	}
	if f.DeviceName != "my-device" {
		t.Errorf("Expected DeviceName 'my-device', got %q", f.DeviceName)
	}
	if f.Size != int64(len("content")) {
		t.Errorf("Expected Size %d, got %d", len("content"), f.Size)
	}
	if f.DirPath == "" {
		t.Error("Expected non-empty DirPath")
	}
	if f.FullPath == "" {
		t.Error("Expected non-empty FullPath")
	}
}
