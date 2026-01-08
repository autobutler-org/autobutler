package usbutil

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

type Partition interface {
	MountCommand(mountTargetPath string) *exec.Cmd
	MountPath() (string, error)
	Path() string
	SizeBytes() (int, error)
	Stat() (*unix.Statfs_t, error)
}
