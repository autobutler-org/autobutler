package storageutil

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// TODO(pre-v1.0.0, #1601): delete this file and its tests, and drop the
// migrateLegacyCirrusDir call in setupFilesDirIn.
//
// The on-disk storage root was named "cirrus" until the Cirrus -> Files rename
// (#1601). Existing installs still have their files under <dataDir>/cirrus, so
// the directory is migrated to <dataDir>/files on startup.
//
// Scope is the system data dir only. External devices formatted by a pre-rename
// build are deliberately left alone — see GetFilesDirForDevice.
//
// This runs on every startup and must stay idempotent. Once no install predates
// the rename, delete it.
const legacyCirrusDirName = "cirrus"

// migrateLegacyCirrusDir moves <dataDir>/cirrus to <dataDir>/files.
//
// The common case is a plain rename, which is atomic within a filesystem. When
// both directories exist — an install that was downgraded and re-upgraded, say
// — the legacy entries are merged into the target under non-conflicting names
// rather than left behind, because an orphaned cirrus directory hides a user's
// files from the UI without telling them.
//
// It is a no-op when there is nothing to migrate, so it is safe to call on
// every startup.
func migrateLegacyCirrusDir(dataDir string) error {
	legacyDir := filepath.Join(dataDir, legacyCirrusDirName)
	targetDir := ConstructFilesDir(dataDir)

	if legacyDir == targetDir {
		// Only reachable if ConstructFilesDir is repointed at "cirrus" again.
		// Renaming a directory onto itself would destroy it, so refuse.
		return fmt.Errorf(
			"refusing to migrate: legacy and target directories are the same path (%s)", legacyDir)
	}

	legacyInfo, err := os.Stat(legacyDir)
	if os.IsNotExist(err) {
		// Already migrated, or a fresh install. Nothing to do.
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat legacy cirrus directory: %w", err)
	}
	if !legacyInfo.IsDir() {
		// Something else owns that name. Not ours to move.
		slog.Warn("legacy cirrus path is not a directory, skipping migration", "path", legacyDir)
		return nil
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// The common path: no target yet, so hand the whole directory over in
		// one atomic rename. Nothing is copied and nothing can be half-moved.
		if err := os.Rename(legacyDir, targetDir); err != nil {
			return fmt.Errorf("failed to rename %s to %s: %w", legacyDir, targetDir, err)
		}
		slog.Info("migrated legacy cirrus directory", "from", legacyDir, "to", targetDir)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to stat files directory: %w", err)
	}

	// Both exist. Merge rather than strand the legacy directory.
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		return fmt.Errorf("failed to read legacy cirrus directory: %w", err)
	}
	slog.Warn(
		"both cirrus and files directories exist, merging legacy entries",
		"legacy", legacyDir,
		"target", targetDir,
		"entries", len(entries),
	)
	for _, entry := range entries {
		oldPath := filepath.Join(legacyDir, entry.Name())
		newPath := GetNonConflictingPath(filepath.Join(targetDir, entry.Name()))
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s into the files directory: %w", entry.Name(), err)
		}
		if newPath != filepath.Join(targetDir, entry.Name()) {
			slog.Warn("renamed conflicting entry during migration", "from", oldPath, "to", newPath)
		}
	}
	if err := os.RemoveAll(legacyDir); err != nil {
		return fmt.Errorf("failed to remove the emptied legacy cirrus directory: %w", err)
	}
	return nil
}
