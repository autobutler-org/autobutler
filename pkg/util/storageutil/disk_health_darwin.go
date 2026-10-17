//go:build darwin

// cspell:words smartctl

package storageutil

import (
	"bytes"
	"context"
)

// defaultHelperInstalled is false on macOS. The root helper is a Linux
// smartctl wrapper; darwin reads the SMART status diskutil info already prints.
func defaultHelperInstalled() bool { return false }

func readPlatform(ctx context.Context, devicePath string, system HealthSystem) DiskHealth {
	if _, err := system.LookPath("diskutil"); err != nil {
		return toolMissingHealth()
	}
	out, err := system.Run(ctx, "diskutil", "info", devicePath)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return UnknownDiskHealth()
	}
	return classifyDiskutilInfo(string(out))
}
