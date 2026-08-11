package v0_devices

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autobutler-org/autobutler/internal/db"
	_ "modernc.org/sqlite"
)

// schema is the DDL for the registered_devices table + indexes, copied from
// migration 020_device_registration.up.sql so tests are self-contained.
const registrationSchema = `
CREATE TABLE IF NOT EXISTS registered_devices (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL DEFAULT '',
    device_type     TEXT NOT NULL DEFAULT 'unknown',
    identity_type   TEXT NOT NULL DEFAULT 'local',
    ip_address      TEXT NOT NULL DEFAULT '',
    mac_address     TEXT,
    tailscale_key   TEXT,
    user_agent      TEXT NOT NULL DEFAULT '',
    approval_status TEXT NOT NULL DEFAULT 'pending',
    approved_by     TEXT,
    approved_at     DATETIME,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_registered_devices_ip_ua
    ON registered_devices (ip_address, user_agent) WHERE tailscale_key IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_registered_devices_ts_key
    ON registered_devices (tailscale_key) WHERE tailscale_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_registered_devices_status
    ON registered_devices (approval_status);
`

func newTestRegistrationDB(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if _, err := sqlDB.Exec(registrationSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	return db.New(conn)
}

func TestRegisterDevice_NewDevice(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	d, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
		Name:         "Alice's Phone",
		DeviceType:   "phone",
		IdentityType: "local",
		IpAddress:    "192.168.1.42",
		MacAddress:   sql.NullString{String: "aa:bb:cc:dd:ee:ff", Valid: true},
		TailscaleKey: sql.NullString{},
		UserAgent:    "AutoButler/1.0 (Android)",
	})
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	if d.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if d.Name != "Alice's Phone" {
		t.Errorf("name: got %q, want %q", d.Name, "Alice's Phone")
	}
	if d.ApprovalStatus != "pending" {
		t.Errorf("initial status: got %q, want \"pending\"", d.ApprovalStatus)
	}
	if !d.MacAddress.Valid || d.MacAddress.String != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("mac: got %v, want aa:bb:cc:dd:ee:ff", d.MacAddress)
	}
}

func TestRegisterDevice_Idempotent(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	params := db.RegisterDeviceParams{
		Name:         "Bob's Laptop",
		DeviceType:   "laptop",
		IdentityType: "local",
		IpAddress:    "192.168.1.99",
		MacAddress:   sql.NullString{},
		TailscaleKey: sql.NullString{},
		UserAgent:    "AutoButler/2.0 (Linux)",
	}

	d1, err := q.RegisterDevice(ctx, params)
	if err != nil {
		t.Fatalf("first RegisterDevice: %v", err)
	}

	// Re-register same IP+UA: should upsert, not error.
	params.Name = "Bob's Laptop (renamed)"
	d2, err := q.RegisterDevice(ctx, params)
	if err != nil {
		t.Fatalf("second RegisterDevice: %v", err)
	}

	if d1.ID != d2.ID {
		t.Errorf("IDs differ on upsert: %d vs %d", d1.ID, d2.ID)
	}
	if d2.Name != "Bob's Laptop (renamed)" {
		t.Errorf("name not updated: got %q", d2.Name)
	}
}

func TestApproveDevice(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	d, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
		Name:         "Carol's Tablet",
		DeviceType:   "tablet",
		IdentityType: "local",
		IpAddress:    "10.0.0.5",
		MacAddress:   sql.NullString{},
		TailscaleKey: sql.NullString{},
		UserAgent:    "AutoButler/1.0 (iOS)",
	})
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	approved, err := q.ApproveDevice(ctx, db.ApproveDeviceParams{
		ApprovedBy: sql.NullString{String: "admin", Valid: true},
		ID:         d.ID,
	})
	if err != nil {
		t.Fatalf("ApproveDevice: %v", err)
	}

	if approved.ApprovalStatus != "approved" {
		t.Errorf("status: got %q, want \"approved\"", approved.ApprovalStatus)
	}
	if !approved.ApprovedBy.Valid || approved.ApprovedBy.String != "admin" {
		t.Errorf("approved_by: got %v, want \"admin\"", approved.ApprovedBy)
	}
	if !approved.ApprovedAt.Valid {
		t.Error("expected approved_at to be set")
	}
}

func TestRevokeDevice(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	d, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
		Name:         "Dave's PC",
		DeviceType:   "laptop",
		IdentityType: "local",
		IpAddress:    "172.16.0.10",
		MacAddress:   sql.NullString{},
		TailscaleKey: sql.NullString{},
		UserAgent:    "AutoButler/1.0 (Windows)",
	})
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	revoked, err := q.RevokeDevice(ctx, db.RevokeDeviceParams{
		ApprovedBy: sql.NullString{String: "admin", Valid: true},
		ID:         d.ID,
	})
	if err != nil {
		t.Fatalf("RevokeDevice: %v", err)
	}

	if revoked.ApprovalStatus != "revoked" {
		t.Errorf("status: got %q, want \"revoked\"", revoked.ApprovalStatus)
	}
}

func TestListRegisteredDevicesByStatus(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	// Register two devices
	for _, ua := range []string{"UA-1", "UA-2"} {
		_, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
			Name:         "device",
			DeviceType:   "phone",
			IdentityType: "local",
			IpAddress:    "1.2.3.4",
			MacAddress:   sql.NullString{},
			TailscaleKey: sql.NullString{},
			UserAgent:    ua,
		})
		if err != nil {
			t.Fatalf("RegisterDevice(%s): %v", ua, err)
		}
	}

	pending, err := q.ListRegisteredDevicesByStatus(ctx, "pending")
	if err != nil {
		t.Fatalf("ListRegisteredDevicesByStatus: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending devices, got %d", len(pending))
	}

	// Approve first
	_, err = q.ApproveDevice(ctx, db.ApproveDeviceParams{
		ApprovedBy: sql.NullString{String: "admin", Valid: true},
		ID:         pending[0].ID,
	})
	if err != nil {
		t.Fatalf("ApproveDevice: %v", err)
	}

	pending2, err := q.ListRegisteredDevicesByStatus(ctx, "pending")
	if err != nil {
		t.Fatalf("ListRegisteredDevicesByStatus after approve: %v", err)
	}
	if len(pending2) != 1 {
		t.Errorf("expected 1 pending device after approve, got %d", len(pending2))
	}
}

func TestCountPendingDevices(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	count, err := q.CountPendingDevices(ctx)
	if err != nil {
		t.Fatalf("CountPendingDevices: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	_, err = q.RegisterDevice(ctx, db.RegisterDeviceParams{
		Name: "x", DeviceType: "phone", IdentityType: "local",
		IpAddress: "9.9.9.9", MacAddress: sql.NullString{},
		TailscaleKey: sql.NullString{}, UserAgent: "ua",
	})
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	count, err = q.CountPendingDevices(ctx)
	if err != nil {
		t.Fatalf("CountPendingDevices: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestDeleteRegisteredDevice(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	d, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
		Name: "temp", DeviceType: "phone", IdentityType: "local",
		IpAddress: "8.8.8.8", MacAddress: sql.NullString{},
		TailscaleKey: sql.NullString{}, UserAgent: "ua-temp",
	})
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	if err := q.DeleteRegisteredDevice(ctx, d.ID); err != nil {
		t.Fatalf("DeleteRegisteredDevice: %v", err)
	}

	_, err = q.GetRegisteredDevice(ctx, d.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows after delete, got: %v", err)
	}
}

func TestRegisterTailscaleDevice(t *testing.T) {
	q := newTestRegistrationDB(t)
	ctx := context.Background()

	tsKey := "nodekey:abc123"
	d, err := q.RegisterTailscaleDevice(ctx, db.RegisterTailscaleDeviceParams{
		Name:         "Eve's Mac",
		DeviceType:   "laptop",
		IpAddress:    "100.64.0.5",
		TailscaleKey: sql.NullString{String: tsKey, Valid: true},
		UserAgent:    "AutoButler/1.0 (macOS)",
	})
	if err != nil {
		t.Fatalf("RegisterTailscaleDevice: %v", err)
	}

	if d.IdentityType != "tailscale" {
		t.Errorf("identity_type: got %q, want \"tailscale\"", d.IdentityType)
	}
	if !d.TailscaleKey.Valid || d.TailscaleKey.String != tsKey {
		t.Errorf("tailscale_key: got %v, want %q", d.TailscaleKey, tsKey)
	}
}
