//go:build linux

package usbutil

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestUnmountCommand_BuildsCorrectArgs verifies that UnmountCommand produces a
// "umount <path>" command without calling the real binary.
func TestUnmountCommand_BuildsCorrectArgs(t *testing.T) {
	cmd := UnmountCommand("/mnt/usb")
	if cmd == nil {
		t.Fatal("UnmountCommand returned nil")
	}
	args := cmd.Args
	if len(args) < 2 {
		t.Fatalf("expected at least 2 args, got %v", args)
	}
	if !strings.HasSuffix(args[0], "umount") {
		t.Errorf("expected first arg to be umount binary, got %q", args[0])
	}
	if args[len(args)-1] != "/mnt/usb" {
		t.Errorf("expected last arg to be /mnt/usb, got %q", args[len(args)-1])
	}
}

// TestMountCommand_BuildsCorrectArgs verifies that partition.MountCommand
// produces a "mount <dev> <target>" command on Linux.
func TestMountCommand_BuildsCorrectArgs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	p := &partition{path: "/dev/sda1"}
	cmd := p.MountCommand("/mnt/target")
	if cmd == nil {
		t.Fatal("MountCommand returned nil")
	}
	args := cmd.Args
	if len(args) < 3 {
		t.Fatalf("expected at least 3 args, got %v", args)
	}
	if !strings.HasSuffix(args[0], "mount") {
		t.Errorf("expected mount binary, got %q", args[0])
	}
	if args[len(args)-2] != "/dev/sda1" {
		t.Errorf("expected device arg /dev/sda1, got %q", args[len(args)-2])
	}
	if args[len(args)-1] != "/mnt/target" {
		t.Errorf("expected target arg /mnt/target, got %q", args[len(args)-1])
	}
}

// TestPath_ReturnsDevicePath verifies that partition.Path() returns the exact
// path string given at construction.
func TestPath_ReturnsDevicePath(t *testing.T) {
	p := &partition{path: "/dev/sdb2"}
	if got := p.Path(); got != "/dev/sdb2" {
		t.Errorf("Path() = %q; want /dev/sdb2", got)
	}
}

// TestMountPath_ErrorForUnmountedDevice verifies that MountPath returns an
// error (not a mount point) for a device path that does not appear in
// /proc/mounts.
func TestMountPath_ErrorForUnmountedDevice(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	if _, err := os.Stat("/proc/mounts"); err != nil {
		t.Skip("/proc/mounts not available")
	}
	p := &partition{path: "/dev/nonexistent_device_xyz_test"}
	_, err := p.MountPath()
	if err == nil {
		t.Error("expected error for unmounted device, got nil")
	}
}

// TestMountPath_FindsRootMount verifies that MountPath can locate the root
// device's mount point when the root device string is known from /proc/mounts.
func TestMountPath_FindsRootMount(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		t.Skipf("cannot read /proc/mounts: %v", err)
	}

	// Find any line in /proc/mounts so we have a real device we can look up.
	var devPath, mountPoint string
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.HasPrefix(fields[0], "/dev/") {
			devPath = fields[0]
			mountPoint = fields[1]
			break
		}
	}
	if devPath == "" {
		t.Skip("no /dev/* entry in /proc/mounts to test against")
	}

	p := &partition{path: devPath}
	got, err := p.MountPath()
	if err != nil {
		t.Fatalf("MountPath() for %q: %v", devPath, err)
	}
	if got != mountPoint {
		t.Errorf("MountPath() = %q; want %q", got, mountPoint)
	}
}

// TestUnmountCommand_IsExecCmd verifies that UnmountCommand returns an
// *exec.Cmd (not a stub), confirming the caller can Run() it if needed.
func TestUnmountCommand_IsExecCmd(t *testing.T) {
	cmd := UnmountCommand("/mnt/drive")
	if _, ok := interface{}(cmd).(*exec.Cmd); !ok {
		t.Error("UnmountCommand did not return *exec.Cmd")
	}
}
