//go:build linux
// +build linux

package storageutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

type partition struct {
	path string
}

func (p *partition) MountCommand(mountTargetPath string) *exec.Cmd {
	return exec.Command("sudo", "mount", p.path, mountTargetPath)
}

func UnmountCommand(mountTargetPath string) *exec.Cmd {
	return exec.Command("sudo", "umount", mountTargetPath)
}

func (p *partition) MountPath() (string, error) {
	return findMountPath(p.path)
}

// findMountPath scans /proc/mounts for a device path, resolving symlinks
// on both sides so that /dev/disk/by-* entries match /dev/sdX paths.
func findMountPath(devicePath string) (string, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/mounts: %w", err)
	}

	// Resolve the target device path to its real path for comparison.
	resolvedTarget, err := resolveDevicePath(devicePath)
	if err != nil {
		resolvedTarget = devicePath
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Direct match first (fast path).
		if fields[0] == devicePath {
			return fields[1], nil
		}
		// Resolve symlinks in /proc/mounts entry and compare.
		resolvedMount, err := resolveDevicePath(fields[0])
		if err != nil {
			continue
		}
		if resolvedMount == resolvedTarget {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("device %s is not mounted", devicePath)
}

// resolveDevicePath resolves a /dev path through symlinks to its real path.
// This handles /dev/disk/by-uuid/*, /dev/disk/by-label/*, etc.
func resolveDevicePath(path string) (string, error) {
	if !strings.HasPrefix(path, "/dev/") {
		return path, nil
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Not a symlink or can't resolve — return as-is.
		return path, nil
	}
	return resolved, nil
}

func (p *partition) Path() string {
	return p.path
}

func (p *partition) SizeBytes() (int, error) {
	stat, err := p.Stat()
	if err == nil {
		// If we can stat, use that
		return int(stat.Blocks * uint64(stat.Bsize)), nil
	}

	// If not mounted, use ioctl on the block device
	f, err := os.Open(p.path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	size, err := unix.IoctlGetInt(int(f.Fd()), unix.BLKGETSIZE64)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func (p *partition) Stat() (*unix.Statfs_t, error) {
	mountPath, err := p.MountPath()
	if err != nil {
		return nil, fmt.Errorf("partition %s is not mounted", p.path)
	}

	var stat unix.Statfs_t
	err = unix.Statfs(mountPath, &stat)
	if err != nil {
		return nil, fmt.Errorf("failed to statfs mount path %s: %w", mountPath, err)
	}
	return &stat, nil
}
