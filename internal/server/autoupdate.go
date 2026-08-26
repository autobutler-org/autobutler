package server

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/autobutler-org/quark/pkg/util/updateutil"
	"github.com/autobutler-org/quark/pkg/util/versionutil"
)

// startAutoUpdateChecker runs a background goroutine that checks for and
// installs updates every 24 hours, but only when auto-update is enabled in
// settings. If a newer version is found, it is installed and the process
// exits to allow the system supervisor to restart it.
//
// The goroutine exits when ctx is cancelled (e.g. on graceful server shutdown).
func startAutoUpdateChecker(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[autoupdate] shutting down")
			return
		case <-ticker.C:
		}

		if !settingsutil.GetAutoUpdate() {
			log.Printf("[autoupdate] auto-update is disabled, skipping check")
			continue
		}

		// Nothing downstream can succeed if the binary cannot be replaced, and
		// the check is local — so it runs before any network request rather
		// than after a full download (#1609).
		if err := updateutil.CanSelfUpdate(); err != nil {
			log.Printf("[autoupdate] skipping check: %v", err)
			continue
		}

		log.Printf("[autoupdate] checking for updates...")

		latestVersion, err := updateutil.GetLatestVersionFromDefaultSources()
		if err != nil {
			log.Printf("[autoupdate] failed to get latest version: %v", err)
			continue
		}

		current := versionutil.GetVersion()
		cmp := versionutil.CompareVersions(
			versionutil.Version{Semver: latestVersion},
			*current,
		)

		// cmp == 2 means one or both versions are NOSEMVER or dev- prefixed;
		// skip to avoid auto-updating dev/untagged builds.
		if cmp <= 0 || cmp == 2 {
			log.Printf("[autoupdate] already up to date or version indeterminate (current: %s, latest: %s, cmp: %d)", current.Semver, latestVersion, cmp)
			continue
		}

		log.Printf("[autoupdate] newer version available: %s (current: %s), installing...", latestVersion, current.Semver)

		if err := updateutil.UpdateFromDefaultSources(latestVersion); err != nil {
			log.Printf("[autoupdate] update failed: %v", err)
			continue
		}

		log.Printf("[autoupdate] update to %s succeeded, restarting...", latestVersion)
		os.Exit(0)
	}
}
