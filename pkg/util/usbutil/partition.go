package usbutil

import (
	"os/exec"
)

type Partition interface {
	MountCommand(mountTargetPath string) *exec.Cmd
	MountPath() (string, error)
	Path() string
	SizeBytes() (int, error)
}

type partition struct {
	path string
}

func UnmountCommand(mountTargetPath string) *exec.Cmd {
	return exec.Command("umount", mountTargetPath)
}
