package backup

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/storageutil"

	_ "modernc.org/sqlite"
)

// newRolesQueries returns queries over an in-memory device_roles table, which
// is all StartSnapshotBackup reads before it decides whether to go on.
func newRolesQueries(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.Exec(`
		CREATE TABLE device_roles (
			device_serial TEXT PRIMARY KEY,
			role TEXT NOT NULL DEFAULT 'unassigned',
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("open connection: %v", err)
	}
	return db.New(conn)
}

func TestStartSnapshotBackup_RejectsDeviceWithoutRole(t *testing.T) {
	const serial = "USB-001"
	ctx := context.Background()
	queries := newRolesQueries(t)
	if err := queries.UpsertDeviceRole(ctx, db.UpsertDeviceRoleParams{
		DeviceSerial: serial,
		Role:         "default-storage",
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}

	_, err := StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx:                ctx,
		Queries:            queries,
		Store:              NewInMemoryBackupJobStore(),
		TargetDeviceSerial: serial,
	})
	if !errors.Is(err, ErrTargetRoleRequired) {
		t.Fatalf("expected ErrTargetRoleRequired, got %v", err)
	}
}

func TestStartSnapshotBackup_RejectsUnknownDevice(t *testing.T) {
	_, err := StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx:                context.Background(),
		Queries:            newRolesQueries(t),
		Store:              NewInMemoryBackupJobStore(),
		TargetDeviceSerial: "NOT-A-DEVICE",
	})
	if !errors.Is(err, ErrTargetRoleRequired) {
		t.Fatalf("expected ErrTargetRoleRequired, got %v", err)
	}
}

func TestStartSnapshotBackup_RejectsSecondBackupForSameTarget(t *testing.T) {
	const serial = "USB-001"
	ctx := context.Background()
	queries := newRolesQueries(t)
	if err := queries.UpsertDeviceRole(ctx, db.UpsertDeviceRoleParams{
		DeviceSerial: serial,
		Role:         "snapshot-backup",
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}

	store := NewInMemoryBackupJobStore()
	running := &BackupJob{ID: "job-1", Status: BackupStatusCopying, TargetDeviceSerial: serial}
	if err := store.Create(ctx, running); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	_, err := StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx:                ctx,
		Queries:            queries,
		Store:              store,
		TargetDeviceSerial: serial,
	})
	var inProgress *BackupInProgressError
	if !errors.As(err, &inProgress) {
		t.Fatalf("expected BackupInProgressError, got %v", err)
	}
	if inProgress.JobID != "job-1" {
		t.Errorf("expected the running job id, got %q", inProgress.JobID)
	}
	if want := "backup already running for this device (job job-1)"; inProgress.Error() != want {
		t.Errorf("expected %q, got %q", want, inProgress.Error())
	}
}

func TestStartSnapshotBackup_FinishedJobDoesNotBlockNewOne(t *testing.T) {
	const serial = "USB-001"
	ctx := context.Background()
	queries := newRolesQueries(t)
	if err := queries.UpsertDeviceRole(ctx, db.UpsertDeviceRoleParams{
		DeviceSerial: serial,
		Role:         "snapshot-backup",
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}

	store := NewInMemoryBackupJobStore()
	for _, status := range []BackupJobStatus{BackupStatusCompleted, BackupStatusFailed} {
		if err := store.Create(ctx, &BackupJob{
			ID:                 string(status),
			Status:             status,
			TargetDeviceSerial: serial,
		}); err != nil {
			t.Fatalf("seed job: %v", err)
		}
	}

	// No device is plugged in, so this stops at the device lookup — past the
	// conflict scan, which is the point.
	_, err := StartSnapshotBackup(StartSnapshotBackupParams{
		Ctx:                ctx,
		Queries:            queries,
		Storage:            storageutil.NewStorageService(noDevices{}),
		Store:              store,
		TargetDeviceSerial: serial,
	})
	if !errors.Is(err, ErrTargetNotManaged) {
		t.Fatalf("a finished job should not block a new backup, got %v", err)
	}
}

// noDevices is a Detector with nothing plugged in.
type noDevices struct{}

func (noDevices) DetectDevices() ([]storageutil.Device, error) { return nil, nil }
