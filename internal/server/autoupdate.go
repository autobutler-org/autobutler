package server

import (
	"log"
	"os"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"
	"github.com/autobutler-org/autobutler/pkg/util/updateutil"
	"github.com/autobutler-org/autobutler/pkg/util/versionutil"
)

// startAutoUpdateChecker runs a background goroutine that checks for and
// installs updates every 24 hours, but only when auto-update is enabled in
// settings. If a newer version is found, it is installed and the process
// exits to allow the system supervisor to restart it.
func startAutoUpdateChecker() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if !settingsutil.GetAutoUpdate() {
			log.Printf("[autoupdate] auto-update is disabled, skipping check")
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

		if cmp <= 0 {
			log.Printf("[autoupdate] already up to date (current: %s, latest: %s)", current.Semver, latestVersion)
			continue
		}

		log.Printf("[autoupdate] newer version available: %s (current: %s), installing...", latestVersion, current.Semver)

		if err := updateutil.UpdateFromDefaultSources(latestVersion); err != nil {
			log.Printf("[autoupdate] update failed: %v", err)
			continue
		}

		log.Printf("[autoupdate] update to %s succeeded, restarting...", latestVersion)
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}
}
