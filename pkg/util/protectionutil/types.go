// cspell:ignore protectionutil

package protectionutil

import "time"

const roleSnapshotBackup = "snapshot-backup"

// backupDrive is one snapshot-backup role joined with its device and the
// newest completed job the caller supplied.
type backupDrive struct {
	serial       string
	name         string
	internal     bool
	completedAt  time.Time
	hasCompleted bool
	diskKey      string
	qualifies    bool
	replaceSoon  bool
}
