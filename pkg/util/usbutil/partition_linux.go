//go:build linux
// +build linux

package usbutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

func (p *partition) MountCommand(mountTargetPath string) *exec.Cmd {
	return exec.Command("mount", p.path, mountTargetPath)
}

func (p *partition) MountPath() (string, error) {
	// Check /proc/mounts for this device
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/mounts: %w", err)
	}
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == p.path {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("partition %s is not mountable", p.path)
}

func (p *partition) Path() string {
	return p.path
}

func (p *partition) SizeBytes() (int, error) {
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
