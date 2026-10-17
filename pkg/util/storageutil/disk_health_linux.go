//go:build linux

// cspell:words smartctl

package storageutil

import (
	"context"
	"os"
)

// defaultHelperInstalled reports whether quark install has written the
// smartctl helper. A missing helper is not an error: the read falls through
// to smartctl on PATH, and a missing smartctl is an Unknown verdict.
func defaultHelperInstalled() bool {
	info, err := os.Stat(HelperPath)
	return err == nil && !info.IsDir()
}

func readPlatform(ctx context.Context, devicePath string, system HealthSystem) DiskHealth {
	return readSmartctl(ctx, devicePath, system)
}
