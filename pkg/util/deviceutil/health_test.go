package deviceutil

import (
	"context"
	"errors"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

type healthDetector struct {
	devices []storageutil.Device
}

func (d healthDetector) DetectDevices() ([]storageutil.Device, error) {
	return d.devices, nil
}

type serialUSB struct{ serial string }

func (s serialUSB) GetPath() string                 { return "" }
func (s serialUSB) GetVendorID() string             { return "" }
func (s serialUSB) GetProductID() string            { return "" }
func (s serialUSB) GetManufacturer() string         { return "" }
func (s serialUSB) GetProduct() string              { return "" }
func (s serialUSB) GetSerial() string               { return s.serial }
func (s serialUSB) GetMountPath() string            { return "" }
func (s serialUSB) BlockDevicePath() (string, bool) { return "", false }
func (s serialUSB) IsStorageDevice() bool           { return true }
func (s serialUSB) Partitions() ([]storageutil.Partition, error) {
	return nil, nil
}

func healthDevice(mount, serial, devicePath string) storageutil.Device {
	return storageutil.Device{
		Name:       serial,
		MountPoint: mount,
		DevicePath: devicePath,
		UsbInfo:    serialUSB{serial: serial},
	}
}

func verdictHealth(verdict, reason string) func(storageutil.Device) *storageutil.DiskHealth {
	return func(device storageutil.Device) *storageutil.DiskHealth {
		got := verdict
		why := reason
		if device.UsbInfo != nil && device.UsbInfo.GetSerial() == "GOOD" {
			got = storageutil.HealthHealthy
			why = storageutil.ReasonHealthy
		}
		health := storageutil.DiskHealth{Verdict: got, Reason: why}
		return &health
	}
}

func adminForRole(t *testing.T) db.Querier {
	t.Helper()
	database := dbtest.NewDB(t)
	keyHash, err := authutil.HashKey(dbtest.AuthKey("mypassword"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Queries.CreateUser(context.Background(), db.CreateUserParams{
		Username:    "admin",
		AuthKeyHash: keyHash,
	}); err != nil {
		t.Fatal(err)
	}
	return database.Queries
}

func TestSetRole_ReplaceSoonNeedsAcknowledgment(t *testing.T) {
	queries := adminForRole(t)
	mount := t.TempDir()
	params := SetRoleParams{
		Ctx:      context.Background(),
		Queries:  queries,
		Storage:  storageutil.NewStorageService(healthDetector{devices: []storageutil.Device{healthDevice(mount, "WORN", "/dev/sdb1")}}),
		Serial:   "WORN",
		Role:     "default-storage",
		Username: "admin",
		Password: dbtest.AuthKey("mypassword"),
		Health:   verdictHealth(storageutil.HealthReplaceSoon, storageutil.ReasonReplaceFailed),
	}
	_, err := SetRole(params)
	var needsAck *AcknowledgmentRequiredError
	if !errors.As(err, &needsAck) || needsAck.Stronger {
		t.Fatalf("SetRole without acknowledgment = %v, want acknowledgment required", err)
	}
	if err.Error() != ReplaceSoonAcknowledgment {
		t.Fatalf("error = %q", err.Error())
	}
	role, roleErr := queries.GetDeviceRole(context.Background(), "WORN")
	if roleErr == nil && role == "default-storage" {
		t.Fatal("role was written without acknowledgment")
	}

	params.AcknowledgeHealth = true
	if _, err := SetRole(params); err != nil {
		t.Fatalf("SetRole with acknowledgment = %v", err)
	}
	role, err = queries.GetDeviceRole(context.Background(), "WORN")
	if err != nil || role != "default-storage" {
		t.Fatalf("role = %q, %v", role, err)
	}
}

func TestSetRole_ReplaceSoonBackupAndStrongerDefault(t *testing.T) {
	queries := adminForRole(t)
	ctx := context.Background()
	if err := queries.UpsertDeviceRole(ctx, db.UpsertDeviceRoleParams{DeviceSerial: "GOOD", Role: "default-storage"}); err != nil {
		t.Fatal(err)
	}
	mountA, mountB := t.TempDir(), t.TempDir()
	params := SetRoleParams{
		Ctx:     ctx,
		Queries: queries,
		Storage: storageutil.NewStorageService(healthDetector{devices: []storageutil.Device{
			healthDevice(mountA, "GOOD", "/dev/sda1"),
			healthDevice(mountB, "WORN", "/dev/sdb1"),
		}}),
		Serial:   "WORN",
		Role:     "snapshot-backup",
		Username: "admin",
		Password: dbtest.AuthKey("mypassword"),
		Health:   verdictHealth(storageutil.HealthReplaceSoon, storageutil.ReasonReplaceFailed),
	}
	_, err := SetRole(params)
	var needsAck *AcknowledgmentRequiredError
	if !errors.As(err, &needsAck) || needsAck.Stronger {
		t.Fatalf("backup role = %v, want a plain acknowledgment error", err)
	}

	params.Role = "default-storage"
	_, err = SetRole(params)
	if !errors.As(err, &needsAck) || !needsAck.Stronger || err.Error() != ReplaceSoonWorseAcknowledgment {
		t.Fatalf("worse default = %v, want the stronger acknowledgment", err)
	}

	params.AcknowledgeHealth = true
	if _, err := SetRole(params); err != nil {
		t.Fatalf("acknowledged worse default = %v", err)
	}
}

func TestSetRole_WatchAndUnknownDoNotBlock(t *testing.T) {
	for _, verdict := range []string{storageutil.HealthWatch, storageutil.HealthUnknown} {
		t.Run(verdict, func(t *testing.T) {
			queries := adminForRole(t)
			params := SetRoleParams{
				Ctx:      context.Background(),
				Queries:  queries,
				Storage:  storageutil.NewStorageService(healthDetector{devices: []storageutil.Device{healthDevice(t.TempDir(), "DISK", "/dev/sdc1")}}),
				Serial:   "DISK",
				Role:     "snapshot-backup",
				Username: "admin",
				Password: dbtest.AuthKey("mypassword"),
				Health:   verdictHealth(verdict, "reason"),
			}
			if _, err := SetRole(params); err != nil {
				t.Fatalf("SetRole(%s) = %v", verdict, err)
			}
			if verdict == storageutil.HealthHealthy {
				t.Fatal("this case must not be the healthy verdict")
			}
		})
	}
}

func TestRefuseReplaceSoonTarget(t *testing.T) {
	storage := storageutil.NewStorageService(healthDetector{devices: []storageutil.Device{
		healthDevice(t.TempDir(), "WORN", "/dev/sdb"),
	}})
	params := BackupHealthParams{
		Ctx:     context.Background(),
		Storage: storage,
		Serial:  "WORN",
		Health:  verdictHealth(storageutil.HealthReplaceSoon, storageutil.ReasonReplaceFailed),
	}
	if err := RefuseReplaceSoonTarget(params); !errors.As(err, new(*AcknowledgmentRequiredError)) {
		t.Fatalf("backup without acknowledgment = %v", err)
	}
	params.AcknowledgeHealth = true
	if err := RefuseReplaceSoonTarget(params); err != nil {
		t.Fatalf("acknowledged backup = %v", err)
	}
	params.AcknowledgeHealth = false
	params.Health = verdictHealth(storageutil.HealthWatch, storageutil.ReasonWatchHours)
	if err := RefuseReplaceSoonTarget(params); err != nil {
		t.Fatalf("watch backup = %v", err)
	}
}

func TestListStatuses_MissingToolStillLists(t *testing.T) {
	result, err := ListStatuses(ListStatusesParams{
		Ctx:     context.Background(),
		Storage: storageutil.NewStorageService(healthDetector{devices: []storageutil.Device{healthDevice(t.TempDir(), "DISK", "/dev/sdb")}}),
		Health: func(context.Context, storageutil.Device) *storageutil.DiskHealth {
			health := storageutil.DiskHealth{Verdict: storageutil.HealthUnknown, Reason: storageutil.ReasonToolMissing}
			return &health
		},
	})
	if err != nil {
		t.Fatalf("ListStatuses = %v", err)
	}
	if len(result.Statuses) != 1 || result.Statuses[0].Health == nil {
		t.Fatalf("statuses = %+v", result.Statuses)
	}
	health := result.Statuses[0].Health
	if health.Verdict != storageutil.HealthUnknown || health.Verdict == storageutil.HealthHealthy {
		t.Fatalf("verdict = %q", health.Verdict)
	}
	if health.Reason != storageutil.ReasonToolMissing {
		t.Fatalf("reason = %q", health.Reason)
	}
}

func TestListStatuses_WorseThanDefault(t *testing.T) {
	database := dbtest.NewDB(t)
	ctx := context.Background()
	if err := database.Queries.UpsertDeviceRole(ctx, db.UpsertDeviceRoleParams{DeviceSerial: "GOOD", Role: "default-storage"}); err != nil {
		t.Fatal(err)
	}
	result, err := ListStatuses(ListStatusesParams{
		Ctx:      ctx,
		Storage:  storageutil.NewStorageService(healthDetector{devices: []storageutil.Device{healthDevice(t.TempDir(), "GOOD", "/dev/sda"), healthDevice(t.TempDir(), "WORN", "/dev/sdb")}}),
		Database: database,
		Health: func(_ context.Context, device storageutil.Device) *storageutil.DiskHealth {
			verdict := storageutil.HealthHealthy
			reason := storageutil.ReasonHealthy
			if device.UsbInfo != nil && device.UsbInfo.GetSerial() == "WORN" {
				verdict = storageutil.HealthWatch
				reason = storageutil.ReasonWatchHours
			}
			health := storageutil.DiskHealth{Verdict: verdict, Reason: reason}
			return &health
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	bySerial := map[string]*storageutil.DiskHealth{}
	for _, status := range result.Statuses {
		bySerial[status.UsbInfo.GetSerial()] = status.Health
	}
	if bySerial["GOOD"] == nil || bySerial["GOOD"].WorseThanDefault {
		t.Fatalf("current default = %+v", bySerial["GOOD"])
	}
	if bySerial["WORN"] == nil || !bySerial["WORN"].WorseThanDefault || bySerial["WORN"].Verdict != storageutil.HealthWatch {
		t.Fatalf("watch drive = %+v", bySerial["WORN"])
	}
}
