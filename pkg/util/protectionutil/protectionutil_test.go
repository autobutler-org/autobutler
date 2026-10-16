// cspell:ignore mmcblk nvme protectionutil sda sdb sdc

package protectionutil_test

import (
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/protectionutil"
)

func TestStatus_NoBackupDrive(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	// The live copy is the default-storage drive. A completed job aimed at it
	// is still not a backup copy.
	got := protectionutil.Status(protectionutil.StatusParams{
		Now:   now,
		Roles: []protectionutil.Role{{Serial: "", Role: "default-storage"}},
		Devices: []protectionutil.Device{{
			Serial: "", Name: "Internal", DevicePath: "/dev/mmcblk0p2", IsInternal: true,
		}},
		Completions: []protectionutil.Completion{{
			Serial: "", CompletedAt: now.Add(-time.Hour),
		}},
	})
	if got.Level != protectionutil.LevelUnprotected {
		t.Errorf("level = %s, want unprotected", got.Level)
	}
	if got.NextStep != protectionutil.NextStepAssignRole {
		t.Errorf("next = %s, want assign-backup-role", got.NextStep)
	}
	if len(got.Drives) != 0 {
		t.Errorf("drives = %+v, want none", got.Drives)
	}
	const want = "Your files are only on this Quark. Assign the backup role to another drive."
	if got.Sentence != want {
		t.Errorf("sentence = %q, want %q", got.Sentence, want)
	}
}

func TestStatus_OneGoodDrive(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-protectionutil.StaleAfter + time.Second)
	got := protectionutil.Status(protectionutil.StatusParams{
		Now: now,
		Roles: []protectionutil.Role{
			{Serial: "", Role: "default-storage"},
			{Serial: "usb-a", Role: "snapshot-backup"},
		},
		Devices: []protectionutil.Device{
			{Serial: "", Name: "Internal", DevicePath: "/dev/mmcblk0p2", IsInternal: true},
			{Serial: "usb-a", Name: "Backup", DevicePath: "/dev/sdb1"},
		},
		Completions: []protectionutil.Completion{
			{Serial: "usb-a", CompletedAt: now.Add(-protectionutil.StaleAfter - time.Hour)},
			{Serial: "usb-a", CompletedAt: fresh},
		},
	})
	if got.Level != protectionutil.LevelBasic {
		t.Errorf("level = %s, want basic", got.Level)
	}
	if got.NextStep != protectionutil.NextStepAssignRole {
		t.Errorf("next = %s, want assign-backup-role", got.NextStep)
	}
	const want = "Your files are on 2 drives. Add a second backup drive and keep it somewhere else."
	if got.Sentence != want {
		t.Errorf("sentence = %q, want %q", got.Sentence, want)
	}
	if len(got.Drives) != 1 || !got.Drives[0].Counts || got.Drives[0].Serial != "usb-a" {
		t.Fatalf("drives = %+v, want usb-a counting", got.Drives)
	}
	if got.Drives[0].LastBackedUp == nil || !got.Drives[0].LastBackedUp.Equal(fresh) {
		t.Errorf("last backed up = %v, want %v", got.Drives[0].LastBackedUp, fresh)
	}
}

func TestStatus_TwoGoodDrives(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)
	got := protectionutil.Status(protectionutil.StatusParams{
		Now: now,
		Roles: []protectionutil.Role{
			{Serial: "usb-a", Role: "snapshot-backup"},
			{Serial: "usb-b", Role: "snapshot-backup"},
		},
		Devices: []protectionutil.Device{
			{Serial: "usb-a", Name: "Shelf", DevicePath: "/dev/sdb1"},
			{Serial: "usb-b", Name: "Office", DevicePath: "/dev/sdc1"},
		},
		Completions: []protectionutil.Completion{
			{Serial: "usb-a", CompletedAt: fresh},
			{Serial: "usb-b", CompletedAt: fresh},
		},
	})
	if got.Level != protectionutil.LevelProtected {
		t.Errorf("level = %s, want protected", got.Level)
	}
	if got.NextStep != protectionutil.NextStepKeepElsewhere {
		t.Errorf("next = %s, want keep-one-elsewhere", got.NextStep)
	}
	const want = "Your files are on 3 drives. Keep one backup drive somewhere else."
	if got.Sentence != want {
		t.Errorf("sentence = %q, want %q", got.Sentence, want)
	}
	if len(got.Drives) != 2 || !got.Drives[0].Counts || !got.Drives[1].Counts {
		t.Fatalf("drives = %+v, want both counting", got.Drives)
	}
}

func TestStatus_InternalDriveIgnored(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)
	got := protectionutil.Status(protectionutil.StatusParams{
		Now: now,
		Roles: []protectionutil.Role{
			{Serial: "builtin", Role: "snapshot-backup"},
			{Serial: "", Role: "snapshot-backup"},
		},
		Devices: []protectionutil.Device{
			{Serial: "builtin", Name: "Builtin", DevicePath: "/dev/nvme0n1p2", IsInternal: true},
		},
		Completions: []protectionutil.Completion{
			{Serial: "builtin", CompletedAt: fresh},
			{Serial: "", CompletedAt: fresh},
		},
	})
	if got.Level != protectionutil.LevelUnprotected {
		t.Errorf("level = %s, want unprotected", got.Level)
	}
	if got.NextStep != protectionutil.NextStepAssignRole {
		t.Errorf("next = %s, want assign-backup-role", got.NextStep)
	}
	if len(got.Drives) != 2 {
		t.Fatalf("drives = %+v, want both internal drives listed", got.Drives)
	}
	for _, drive := range got.Drives {
		if drive.Counts || drive.Reason != protectionutil.ReasonInternal {
			t.Errorf("drive %+v counts or has the wrong reason", drive)
		}
	}
}

func TestStatus_StaleJob(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	// A backup exactly StaleAfter old is stale, matching backup_stale.
	stale := now.Add(-protectionutil.StaleAfter)
	got := protectionutil.Status(protectionutil.StatusParams{
		Now:   now,
		Roles: []protectionutil.Role{{Serial: "usb-a", Role: "snapshot-backup"}},
		Devices: []protectionutil.Device{{
			Serial: "usb-a", Name: "Backup", DevicePath: "/dev/sdb1",
		}},
		Completions: []protectionutil.Completion{{Serial: "usb-a", CompletedAt: stale}},
	})
	if got.Level != protectionutil.LevelUnprotected {
		t.Errorf("level = %s, want unprotected", got.Level)
	}
	if got.NextStep != protectionutil.NextStepStartBackup {
		t.Errorf("next = %s, want start-backup", got.NextStep)
	}
	if len(got.Drives) != 1 || got.Drives[0].Counts || got.Drives[0].Reason != protectionutil.ReasonStale {
		t.Fatalf("drives = %+v, want a stale drive", got.Drives)
	}
	if got.Drives[0].LastBackedUp == nil || !got.Drives[0].LastBackedUp.Equal(stale) {
		t.Errorf("last backed up = %v, want %v", got.Drives[0].LastBackedUp, stale)
	}
}

func TestStatus_FailedJob(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	// A failed job is not a completion, and neither is a zero timestamp.
	got := protectionutil.Status(protectionutil.StatusParams{
		Now:   now,
		Roles: []protectionutil.Role{{Serial: "usb-a", Role: "snapshot-backup"}},
		Devices: []protectionutil.Device{{
			Serial: "usb-a", Name: "Backup", DevicePath: "/dev/sdb1",
		}},
		Completions: []protectionutil.Completion{{Serial: "usb-a", CompletedAt: time.Time{}}},
	})
	if got.Level != protectionutil.LevelUnprotected {
		t.Errorf("level = %s, want unprotected", got.Level)
	}
	if got.NextStep != protectionutil.NextStepStartBackup {
		t.Errorf("next = %s, want start-backup", got.NextStep)
	}
	if len(got.Drives) != 1 || got.Drives[0].Counts || got.Drives[0].Reason != protectionutil.ReasonNoCompletedBackup {
		t.Fatalf("drives = %+v, want no completed backup", got.Drives)
	}
	if got.Drives[0].LastBackedUp != nil {
		t.Errorf("last backed up = %v, want none", got.Drives[0].LastBackedUp)
	}
}

func TestStatus_TwoPartitionsCountOnce(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)

	t.Run("mac", func(t *testing.T) {
		got := protectionutil.Status(twoPartitions(now, fresh, "/dev/disk3s1", "/dev/disk3s5"))
		assertOneDisk(t, got)
	})
	t.Run("linux", func(t *testing.T) {
		got := protectionutil.Status(twoPartitions(now, fresh, "/dev/sda1", "/dev/sda2"))
		assertOneDisk(t, got)
	})
	t.Run("one partition qualifies", func(t *testing.T) {
		got := protectionutil.Status(protectionutil.StatusParams{
			Now: now,
			Roles: []protectionutil.Role{
				{Serial: "part-a", Role: "snapshot-backup"},
				{Serial: "part-b", Role: "snapshot-backup"},
			},
			Devices: []protectionutil.Device{
				{Serial: "part-a", DevicePath: "/dev/sda1"},
				{Serial: "part-b", DevicePath: "/dev/sda2"},
			},
			Completions: []protectionutil.Completion{
				{Serial: "part-a", CompletedAt: fresh},
			},
		})
		if got.Level != protectionutil.LevelBasic {
			t.Fatalf("level = %s, want basic", got.Level)
		}
		if !got.Drives[0].Counts || got.Drives[1].Counts {
			t.Fatalf("drives = %+v, want only part-a to count", got.Drives)
		}
		if got.Drives[1].Reason != protectionutil.ReasonNoCompletedBackup {
			t.Errorf("part-b reason = %q", got.Drives[1].Reason)
		}
	})
}

func TestStatus_EmptyDevicePathStandsAlone(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)
	got := protectionutil.Status(protectionutil.StatusParams{
		Now: now,
		Roles: []protectionutil.Role{
			{Serial: "usb-a", Role: "snapshot-backup"},
			{Serial: "usb-b", Role: "snapshot-backup"},
		},
		Devices: []protectionutil.Device{
			{Serial: "usb-a", Name: "Unplugged A"},
			{Serial: "usb-b", Name: "Unplugged B"},
		},
		Completions: []protectionutil.Completion{
			{Serial: "usb-a", CompletedAt: fresh},
			{Serial: "usb-b", CompletedAt: fresh},
		},
	})
	if got.Level != protectionutil.LevelProtected {
		t.Errorf("level = %s, want protected", got.Level)
	}
	if len(got.Drives) != 2 || !got.Drives[0].Counts || !got.Drives[1].Counts {
		t.Fatalf("drives = %+v, want each empty path to count on its own", got.Drives)
	}
}

func twoPartitions(now, fresh time.Time, pathA, pathB string) protectionutil.StatusParams {
	return protectionutil.StatusParams{
		Now: now,
		Roles: []protectionutil.Role{
			{Serial: "part-b", Role: "snapshot-backup"},
			{Serial: "part-a", Role: "snapshot-backup"},
		},
		Devices: []protectionutil.Device{
			{Serial: "part-a", Name: "A", DevicePath: pathA},
			{Serial: "part-b", Name: "B", DevicePath: pathB},
		},
		Completions: []protectionutil.Completion{
			{Serial: "part-a", CompletedAt: fresh},
			{Serial: "part-b", CompletedAt: fresh},
		},
	}
}

func assertOneDisk(t *testing.T, got protectionutil.StatusResult) {
	t.Helper()
	if got.Level != protectionutil.LevelBasic {
		t.Fatalf("level = %s, want basic", got.Level)
	}
	if got.NextStep != protectionutil.NextStepAssignRole {
		t.Errorf("next = %s, want assign-backup-role", got.NextStep)
	}
	var counting int
	for _, drive := range got.Drives {
		if drive.Counts {
			counting++
			if drive.Serial != "part-a" {
				t.Errorf("counting drive = %s, want part-a", drive.Serial)
			}
			continue
		}
		if drive.Reason != protectionutil.ReasonSameDisk {
			t.Errorf("drive %s reason = %q, want same disk", drive.Serial, drive.Reason)
		}
	}
	if counting != 1 {
		t.Fatalf("counting drives = %d, want 1 (%+v)", counting, got.Drives)
	}
}
