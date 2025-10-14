//go:build !windows
// +build !windows

package util

import (
	"golang.org/x/sys/unix"
)

func getAvailableSpaceInBytes(fileDir string) uint64 {
	var stat unix.Statfs_t
	unix.Statfs(fileDir, &stat)
	return stat.Bavail * uint64(stat.Bsize)
}
