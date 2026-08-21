package v0_devices

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	_ "modernc.org/sqlite"
)

const deviceSchema = `
CREATE TABLE IF NOT EXISTS connected_devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address TEXT NOT NULL,
    user_agent TEXT NOT NULL DEFAULT '',
    first_seen_at DATETIME NOT NULL DEFAULT (datetime('now')),
    last_seen_at DATETIME NOT NULL DEFAULT (datetime('now')),
    request_count INTEGER NOT NULL DEFAULT 1,
    UNIQUE (ip_address, user_agent)
);
`

func newDeviceDB(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if _, err := sqlDB.Exec(deviceSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	return db.New(conn)
}

func upsert(t *testing.T, q *db.Queries, ip, ua string) db.ConnectedDevice {
	t.Helper()
	d, err := q.UpsertConnectedDevice(context.Background(), db.UpsertConnectedDeviceParams{
		IpAddress: ip,
		UserAgent: ua,
	})
	if err != nil {
		t.Fatalf("UpsertConnectedDevice(%s, %s): %v", ip, ua, err)
	}
	return d
}

// --- UpsertConnectedDevice ---

func TestUpsert_NewDevice(t *testing.T) {
	q := newDeviceDB(t)
	d := upsert(t, q, "192.168.1.1", "Mozilla/5.0")
	if d.IpAddress != "192.168.1.1" {
		t.Errorf("ip: got %q", d.IpAddress)
	}
	if d.RequestCount != 1 {
		t.Errorf("requestCount: expected 1, got %d", d.RequestCount)
	}
}

func TestUpsert_SameDevice_IncrementsCount(t *testing.T) {
	q := newDeviceDB(t)
	for i := range 5 {
		d := upsert(t, q, "10.0.0.1", "MyApp/1.0")
		if d.RequestCount != int64(i+1) {
			t.Errorf("pass %d: expected requestCount=%d, got %d", i+1, i+1, d.RequestCount)
		}
	}
}

func TestUpsert_SameIP_DifferentUA_TwoRows(t *testing.T) {
	q := newDeviceDB(t)
	upsert(t, q, "10.0.0.2", "Chrome/100")
	upsert(t, q, "10.0.0.2", "Firefox/99")

	count, err := q.CountConnectedDevices(context.Background())
	if err != nil {
		t.Fatalf("CountConnectedDevices: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 distinct rows, got %d", count)
	}
}

func TestUpsert_SameUA_DifferentIP_TwoRows(t *testing.T) {
	q := newDeviceDB(t)
	upsert(t, q, "192.168.0.1", "curl/7.0")
	upsert(t, q, "192.168.0.2", "curl/7.0")

	count, _ := q.CountConnectedDevices(context.Background())
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}
}

// --- ListConnectedDevices ---

func TestListConnectedDevices_Empty(t *testing.T) {
	q := newDeviceDB(t)
	rows, err := q.ListConnectedDevices(context.Background())
	if err != nil {
		t.Fatalf("ListConnectedDevices: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestListConnectedDevices_OrderedByLastSeen(t *testing.T) {
	q := newDeviceDB(t)
	upsert(t, q, "1.1.1.1", "A")
	upsert(t, q, "2.2.2.2", "B")
	// Upsert 1.1.1.1 again to bump last_seen_at.
	upsert(t, q, "1.1.1.1", "A")

	rows, _ := q.ListConnectedDevices(context.Background())
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Most recently seen should be first.
	if rows[0].IpAddress != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1 first (most recently seen), got %s", rows[0].IpAddress)
	}
}

// --- GetConnectedDevice ---

func TestGetConnectedDevice_ByID(t *testing.T) {
	q := newDeviceDB(t)
	created := upsert(t, q, "10.10.10.10", "TestAgent")

	got, err := q.GetConnectedDevice(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetConnectedDevice: %v", err)
	}
	if got.IpAddress != "10.10.10.10" {
		t.Errorf("expected ip 10.10.10.10, got %s", got.IpAddress)
	}
}

// --- DeleteConnectedDevice ---

func TestDeleteConnectedDevice(t *testing.T) {
	q := newDeviceDB(t)
	d := upsert(t, q, "172.16.0.1", "DeleteMe")

	if err := q.DeleteConnectedDevice(context.Background(), d.ID); err != nil {
		t.Fatalf("DeleteConnectedDevice: %v", err)
	}
	count, _ := q.CountConnectedDevices(context.Background())
	if count != 0 {
		t.Errorf("expected 0 after delete, got %d", count)
	}
}

func TestDeleteConnectedDevice_NotFound_NoError(t *testing.T) {
	q := newDeviceDB(t)
	// Deleting a non-existent ID should not error.
	if err := q.DeleteConnectedDevice(context.Background(), 9999); err != nil {
		t.Errorf("delete non-existent device should not error: %v", err)
	}
}

// --- CountConnectedDevices ---

func TestCountConnectedDevices(t *testing.T) {
	q := newDeviceDB(t)
	for i, args := range [][2]string{
		{"192.0.0.1", "UA1"},
		{"192.0.0.2", "UA2"},
		{"192.0.0.3", "UA3"},
	} {
		upsert(t, q, args[0], args[1])
		count, _ := q.CountConnectedDevices(context.Background())
		if count != int64(i+1) {
			t.Errorf("after %d inserts: expected %d, got %d", i+1, i+1, count)
		}
	}
}
