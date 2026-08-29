//go:build linux

package storageutil

import (
	"os"
	"strings"
	"testing"
)

func TestFindMountPath_RootDevice(t *testing.T) {
	// The root device should always be mounted on Linux.
	// Read /proc/mounts to find the root device path first.
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		t.Fatalf("failed to read /proc/mounts: %v", err)
	}

	var rootDev string
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "/" {
			rootDev = fields[0]
			break
		}
	}
	if rootDev == "" {
		t.Skip("could not find root device in /proc/mounts")
	}

	mountPath, err := findMountPath(rootDev)
	if err != nil {
		t.Fatalf("findMountPath(%q) returned error: %v", rootDev, err)
	}
	if mountPath != "/" {
		t.Errorf("findMountPath(%q) = %q, want /", rootDev, mountPath)
	}
}

func TestFindMountPath_NonexistentDevice(t *testing.T) {
	_, err := findMountPath("/dev/nonexistent_device_xyz")
	if err == nil {
		t.Error("findMountPath for nonexistent device should return error")
	}
}

func TestResolveDevicePath_NonDevPath(t *testing.T) {
	// Non-/dev/ paths should be returned as-is.
	result, err := resolveDevicePath("/tmp/something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/tmp/something" {
		t.Errorf("resolveDevicePath(/tmp/something) = %q, want /tmp/something", result)
	}
}

func TestResolveDevicePath_RegularDevPath(t *testing.T) {
	// A /dev/ path that is not a symlink should be returned as-is.
	// /dev/null always exists and is not a symlink.
	result, err := resolveDevicePath("/dev/null")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/dev/null" {
		t.Errorf("resolveDevicePath(/dev/null) = %q, want /dev/null", result)
	}
}

func TestResolveDevicePath_NonexistentDevPath(t *testing.T) {
	// A nonexistent /dev/ path should be returned as-is (not an error).
	result, err := resolveDevicePath("/dev/nonexistent_xyz_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/dev/nonexistent_xyz_123" {
		t.Errorf("resolveDevicePath returned %q, want /dev/nonexistent_xyz_123", result)
	}
}

func TestPartition_MountPath_UsesResolvedPaths(t *testing.T) {
	// Verify that partition.MountPath() delegates to findMountPath.
	// Use a device that definitely isn't mounted.
	p := &partition{path: "/dev/nonexistent_test_device"}
	_, err := p.MountPath()
	if err == nil {
		t.Error("MountPath for nonexistent device should return error")
	}
	if !strings.Contains(err.Error(), "not mounted") {
		t.Errorf("error should mention 'not mounted', got: %v", err)
	}
}
