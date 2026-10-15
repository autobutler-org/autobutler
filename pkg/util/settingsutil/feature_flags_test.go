package settingsutil_test

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/settingsutil"
)

// TestFeatureFlag_UnsetAndRoundTrip checks a flag nobody set reads as unset,
// and that a stored value survives a reload without touching other settings.
func TestFeatureFlag_UnsetAndRoundTrip(t *testing.T) {
	path := settingsFile(t)
	settingsutil.ResetForTesting(path)
	if _, set, err := settingsutil.GetFeatureFlag("chat"); err != nil || set {
		t.Fatalf("GetFeatureFlag with no file: set=%v err=%v; want unset", set, err)
	}
	if err := settingsutil.SetRemoteAccess(true); err != nil {
		t.Fatal(err)
	}

	for _, enabled := range []bool{false, true} {
		if err := settingsutil.SetFeatureFlag("chat", enabled); err != nil {
			t.Fatalf("SetFeatureFlag(%v): %v", enabled, err)
		}
		settingsutil.ResetForTesting(path)
		got, set, err := settingsutil.GetFeatureFlag("chat")
		if err != nil || !set || got != enabled {
			t.Errorf("after reload: %v set=%v err=%v; want %v", got, set, err, enabled)
		}
		if !settingsutil.GetRemoteAccess() {
			t.Error("setting a flag lost another setting")
		}
	}
}

// TestMigrate_DisabledChatFlagTurnsOn checks an install that stored chat off
// imports with that value gone (#2577), so chat is on, and every other flag
// stays. The old chatEnabled switch and a featureFlags.chat row both go.
func TestMigrate_DisabledChatFlagTurnsOn(t *testing.T) {
	for _, file := range []string{
		`{"remoteAccessEnabled":true,"chatEnabled":false}`,
		`{"remoteAccessEnabled":true,"chatEnabled":true}`,
		`{"remoteAccessEnabled":true}`,
		`{"settingsVersion":1,"remoteAccessEnabled":true,"featureFlags":{"chat":false,"calendar":false}}`,
	} {
		path := settingsFile(t)
		if err := os.WriteFile(path, []byte(file), 0600); err != nil {
			t.Fatal(err)
		}
		settingsutil.ResetForTesting(path)
		if _, set, err := settingsutil.GetFeatureFlag("chat"); err != nil || set {
			t.Errorf("%s: chat still stored (set=%v err=%v); want it dropped so chat is on", file, set, err)
		}
		if !settingsutil.GetRemoteAccess() {
			t.Errorf("%s: remoteAccessEnabled lost", file)
		}
		for _, key := range []string{"chatEnabled", "featureFlags.chat"} {
			if _, err := settingsutil.QueriesForTesting().GetSetting(t.Context(), key); !errors.Is(err, sql.ErrNoRows) {
				t.Errorf("%s: %s survived the migration (err=%v)", file, key, err)
			}
		}
	}

	path := settingsFile(t)
	file := `{"settingsVersion":1,"featureFlags":{"chat":false,"calendar":false}}`
	if err := os.WriteFile(path, []byte(file), 0600); err != nil {
		t.Fatal(err)
	}
	settingsutil.ResetForTesting(path)
	if calendar, set, err := settingsutil.GetFeatureFlag("calendar"); err != nil || !set || calendar {
		t.Errorf("calendar = %v set=%v err=%v; want the other flag left off", calendar, set, err)
	}
}

// TestMigrate_RetiredFlagKeyRemoved checks retiring a flag, one
// dropFeatureFlag entry on the migration list, removes its persisted value on
// import and leaves the other flags alone.
func TestMigrate_RetiredFlagKeyRemoved(t *testing.T) {
	settingsutil.SetMigrationsForTesting(t,
		settingsutil.DropFeatureFlagForTesting("unused"),
		settingsutil.DropFeatureFlagForTesting("retired"),
	)
	path := settingsFile(t)
	file := `{"settingsVersion":1,"featureFlags":{"chat":false,"retired":true}}`
	if err := os.WriteFile(path, []byte(file), 0600); err != nil {
		t.Fatal(err)
	}
	settingsutil.ResetForTesting(path)

	if _, set, err := settingsutil.GetFeatureFlag("retired"); err != nil || set {
		t.Errorf("retired flag still set after load (err=%v)", err)
	}
	if chat, set, _ := settingsutil.GetFeatureFlag("chat"); !set || chat {
		t.Error("retiring one flag changed another")
	}
	if _, err := settingsutil.QueriesForTesting().GetSetting(t.Context(), "featureFlags.retired"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("retired flag still has a row (err=%v)", err)
	}
}
