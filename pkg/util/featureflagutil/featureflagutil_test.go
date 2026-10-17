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

// TestEnabled_DefaultWhenUnsetThenStored checks calendar follows its registry
// default until an admin sets it, and the set value after that.
func TestEnabled_DefaultWhenUnsetThenStored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	settingsutil.ResetForTesting(path)
	if !featureflagutil.Enabled(featureflagutil.Calendar) {
		t.Fatal("calendar should default on")
	}
	if featureflagutil.Enabled("no-such-flag") {
		t.Error("an unknown flag should be off")
	}
	// Chat left beta (#2577). Its key is unknown, so it is off here, and the
	// routes no longer consult the flag.
	if featureflagutil.Enabled("chat") {
		t.Error("the retired chat flag should be off")
	}

	result, err := featureflagutil.SetFlag(featureflagutil.SetFlagParams{Key: featureflagutil.Calendar, Enabled: false})
	if err != nil || result.Feature.Enabled || result.Feature.Key != featureflagutil.Calendar {
		t.Fatalf("SetFlag: %+v, %v", result, err)
	}
	settingsutil.ResetForTesting(path)
	if featureflagutil.Enabled(featureflagutil.Calendar) {
		t.Error("calendar should stay off after a reload")
	}
	list, err := featureflagutil.ListFlags(featureflagutil.ListFlagsParams{})
	if err != nil || len(list.Features) != len(featureflagutil.Flags()) || list.Features[0].Enabled {
		t.Errorf("ListFlags: %+v, %v", list, err)
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
