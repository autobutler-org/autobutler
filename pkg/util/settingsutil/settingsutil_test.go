package settingsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"
)

// settingsFile returns the path used by settingsutil during the test.
func settingsFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "settings.json")
}

func TestGetSetRemoteAccess_RoundTrip(t *testing.T) {
	path := settingsFile(t)
	settingsutil.ResetForTesting(path)

	if err := settingsutil.SetRemoteAccess(true, "tskey-auth-abc123"); err != nil {
		t.Fatalf("SetRemoteAccess: %v", err)
	}

	enabled, key := settingsutil.GetRemoteAccess()
	if !enabled {
		t.Error("expected enabled=true")
	}
	if key != "tskey-auth-abc123" {
		t.Errorf("got key %q, want %q", key, "tskey-auth-abc123")
	}
}

func TestGetSetRemoteAccess_Disable(t *testing.T) {
	path := settingsFile(t)
	settingsutil.ResetForTesting(path)

	if err := settingsutil.SetRemoteAccess(true, "tskey-auth-xyz"); err != nil {
		t.Fatalf("SetRemoteAccess enable: %v", err)
	}
	if err := settingsutil.SetRemoteAccess(false, ""); err != nil {
		t.Fatalf("SetRemoteAccess disable: %v", err)
	}

	enabled, key := settingsutil.GetRemoteAccess()
	if enabled {
		t.Error("expected enabled=false after disable")
	}
	if key != "" {
		t.Errorf("expected empty key after disable, got %q", key)
	}
}

func TestGetRemoteAccess_DefaultsWhenNoFile(t *testing.T) {
	path := settingsFile(t)
	settingsutil.ResetForTesting(path)

	enabled, key := settingsutil.GetRemoteAccess()
	if enabled {
		t.Error("expected enabled=false with no settings file")
	}
	if key != "" {
		t.Errorf("expected empty key with no settings file, got %q", key)
	}
}

func TestGetRemoteAccess_PersistsAcrossReset(t *testing.T) {
	path := settingsFile(t)
	settingsutil.ResetForTesting(path)

	if err := settingsutil.SetRemoteAccess(true, "tskey-persisted"); err != nil {
		t.Fatalf("SetRemoteAccess: %v", err)
	}

	// Simulate a process restart: reset in-memory cache so next read comes from disk.
	settingsutil.ResetForTesting(path)

	enabled, key := settingsutil.GetRemoteAccess()
	if !enabled {
		t.Error("expected enabled=true after reload from disk")
	}
	if key != "tskey-persisted" {
		t.Errorf("got key %q, want %q", key, "tskey-persisted")
	}
}

func TestSettingsFile_Permissions(t *testing.T) {
	path := settingsFile(t)
	settingsutil.ResetForTesting(path)

	if err := settingsutil.SetRemoteAccess(true, "tskey-perms"); err != nil {
		t.Fatalf("SetRemoteAccess: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat settings file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("settings file mode %04o, want 0600", mode)
	}
}
