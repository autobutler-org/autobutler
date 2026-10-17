package featureflagutil_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/featureflagutil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
)

// TestRegistry_EveryFlagIsDeclaredInFull checks every flag carries a key,
// admin copy, and the issue that will remove it, and that no key repeats.
func TestRegistry_EveryFlagIsDeclaredInFull(t *testing.T) {
	seen := map[string]bool{}
	for _, flag := range featureflagutil.Flags() {
		if flag.Key == "" || flag.Label == "" || flag.Description == "" || flag.IntroducedIn == "" {
			t.Errorf("flag %+v is missing a key, label, description or introducedIn", flag)
		}
		if flag.SunsetIssue <= 0 {
			t.Errorf("flag %q has no sunsetIssue; every flag needs the issue that removes it", flag.Key)
		}
		if seen[flag.Key] {
			t.Errorf("flag %q is registered twice", flag.Key)
		}
		seen[flag.Key] = true
	}
}

// TestEnabled_RetiredFlagsAreOff checks a key the registry does not declare
// is off, including the flags that left beta: chat (#2577), the calendar
// (#2610), and slides (#2869). None of them can be stored.
func TestEnabled_RetiredFlagsAreOff(t *testing.T) {
	settingsutil.ResetForTesting(filepath.Join(t.TempDir(), "settings.json"))
	for _, key := range []string{"no-such-flag", "chat", "calendar", "slides"} {
		if featureflagutil.Enabled(key) {
			t.Errorf("%s should be off", key)
		}
		_, err := featureflagutil.SetFlag(featureflagutil.SetFlagParams{Key: key, Enabled: false})
		if !errors.Is(err, featureflagutil.ErrUnknownFlag) {
			t.Errorf("SetFlag(%s) = %v; want ErrUnknownFlag", key, err)
		}
	}
	list, err := featureflagutil.ListFlags(featureflagutil.ListFlagsParams{})
	if err != nil || len(list.Features) != 0 || len(featureflagutil.Flags()) != 0 {
		t.Errorf("ListFlags = %+v, flags %d, err %v; want an empty registry", list, len(featureflagutil.Flags()), err)
	}
}

// TestSetFlag_UnknownKeyWritesNothing checks a key the registry does not
// declare is refused rather than stored.
func TestSetFlag_UnknownKeyWritesNothing(t *testing.T) {
	settingsutil.ResetForTesting(filepath.Join(t.TempDir(), "settings.json"))
	_, err := featureflagutil.SetFlag(featureflagutil.SetFlagParams{Key: "retired", Enabled: true})
	if !errors.Is(err, featureflagutil.ErrUnknownFlag) {
		t.Fatalf("err = %v; want ErrUnknownFlag", err)
	}
	if _, set, _ := settingsutil.GetFeatureFlag("retired"); set {
		t.Error("an unknown key was stored")
	}
}
