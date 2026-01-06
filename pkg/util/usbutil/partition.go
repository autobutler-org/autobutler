package usbutil

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

type Partition interface {
	MountCommand(mountTargetPath string) (string, error)
	MountPath() (string, bool)
	Path() string
	SizeBytes() (int, error)
}

type partition struct {
	path string
}

func (p *partition) MountCommand(mountTargetPath string) (string, error) {
	return fmt.Sprintf("mount %s %s", p.path, mountTargetPath), nil
}

func (p *partition) MountPath() (string, bool) {
	// Check /proc/mounts for this device
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == p.path {
			return fields[1], true
		}
	}
	return "", false
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
