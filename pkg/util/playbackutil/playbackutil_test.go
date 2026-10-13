package playbackutil_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/playbackutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

const (
	alice int64 = 1
	bob   int64 = 2
	clip        = "videos/clip.mp4"
)

func newStore(t *testing.T) vfs.MetadataStore {
	t.Helper()
	return vfs.NewSQLiteMetadataStore(dbtest.NewDB(t).Db)
}

func get(t *testing.T, store vfs.MetadataStore, userID int64, relPath string) playbackutil.State {
	t.Helper()
	result, err := playbackutil.Get(context.Background(), playbackutil.GetParams{Store: store, UserID: userID, RelPath: relPath})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return result.State
}

func save(t *testing.T, store vfs.MetadataStore, userID int64, positionMs, durationMs int64) playbackutil.State {
	t.Helper()
	result, err := playbackutil.Save(context.Background(), playbackutil.SaveParams{
		Store: store, UserID: userID, RelPath: clip, PositionMs: positionMs, DurationMs: durationMs,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	return result.State
}

func setWatched(t *testing.T, store vfs.MetadataStore, userID int64, watched bool) playbackutil.State {
	t.Helper()
	result, err := playbackutil.SetWatched(context.Background(), playbackutil.SetWatchedParams{
		Store: store, UserID: userID, RelPath: clip, Watched: watched,
	})
	if err != nil {
		t.Fatalf("SetWatched: %v", err)
	}
	return result.State
}

func TestGet_NothingStoredIsZero(t *testing.T) {
	if got := get(t, newStore(t), alice, clip); got != (playbackutil.State{}) {
		t.Errorf("Get with nothing stored = %+v, want the zero state", got)
	}
}

func TestSaveThenGet_RestoresPosition(t *testing.T) {
	store := newStore(t)
	saved := save(t, store, alice, 30_000, 100_000)
	if saved.PositionMs != 30_000 || saved.DurationMs != 100_000 || saved.Watched {
		t.Errorf("Save = %+v, want position 30000 of 100000, unwatched", saved)
	}
	if _, err := time.Parse(time.RFC3339, saved.UpdatedAt); err != nil {
		t.Errorf("UpdatedAt = %q, want RFC 3339: %v", saved.UpdatedAt, err)
	}
	// The same file however the path is spelled.
	for _, spelling := range []string{clip, "/" + clip, "videos/../videos/clip.mp4"} {
		if got := get(t, store, alice, spelling); got != saved {
			t.Errorf("Get(%q) = %+v, want %+v", spelling, got, saved)
		}
	}
}

func TestSave_NearEndCountsAsWatched(t *testing.T) {
	store := newStore(t)
	if got := save(t, store, alice, 94_999, 100_000); got.Watched || got.PositionMs != 94_999 {
		t.Errorf("just under the threshold = %+v, want unwatched at 94999", got)
	}
	got := save(t, store, alice, 95_000, 100_000)
	if !got.Watched || got.PositionMs != 0 || got.DurationMs != 100_000 {
		t.Errorf("at the threshold = %+v, want watched at position 0 of 100000", got)
	}
	if stored := get(t, store, alice, clip); stored != got {
		t.Errorf("Get = %+v, want %+v", stored, got)
	}
}

func TestSave_UnknownDurationNeverCountsAsWatched(t *testing.T) {
	if got := save(t, newStore(t), alice, 5_000, 0); got.Watched || got.PositionMs != 5_000 {
		t.Errorf("Save with no duration = %+v, want unwatched at 5000", got)
	}
}

func TestSave_WatchedIsSticky(t *testing.T) {
	store := newStore(t)
	save(t, store, alice, 99_000, 100_000)
	got := save(t, store, alice, 40_000, 100_000)
	if !got.Watched || got.PositionMs != 40_000 {
		t.Errorf("mid-video save after watching = %+v, want watched at 40000", got)
	}
}

func TestSave_NegativeIsInvalid(t *testing.T) {
	store := newStore(t)
	for _, params := range []playbackutil.SaveParams{
		{Store: store, UserID: alice, RelPath: clip, PositionMs: -1, DurationMs: 100},
		{Store: store, UserID: alice, RelPath: clip, PositionMs: 1, DurationMs: -100},
	} {
		if _, err := playbackutil.Save(context.Background(), params); !errors.Is(err, playbackutil.ErrInvalid) {
			t.Errorf("Save(%d of %d) = %v, want ErrInvalid", params.PositionMs, params.DurationMs, err)
		}
	}
	if got := get(t, store, alice, clip); got != (playbackutil.State{}) {
		t.Errorf("state after rejected saves = %+v, want none", got)
	}
}

func TestSetWatched_TrueResetsPositionAndKeepsDuration(t *testing.T) {
	store := newStore(t)
	save(t, store, alice, 30_000, 100_000)
	got := setWatched(t, store, alice, true)
	if !got.Watched || got.PositionMs != 0 || got.DurationMs != 100_000 {
		t.Errorf("SetWatched(true) = %+v, want watched at 0 of 100000", got)
	}
}

func TestSetWatched_FalseClearsEverything(t *testing.T) {
	store := newStore(t)
	save(t, store, alice, 99_000, 100_000)
	save(t, store, alice, 40_000, 100_000)
	if got := setWatched(t, store, alice, false); got != (playbackutil.State{}) {
		t.Errorf("SetWatched(false) = %+v, want the zero state", got)
	}
	if got := get(t, store, alice, clip); got != (playbackutil.State{}) {
		t.Errorf("Get after marking unwatched = %+v, want the zero state", got)
	}
}

func TestState_IsPerUser(t *testing.T) {
	store := newStore(t)
	aliceState := save(t, store, alice, 30_000, 100_000)
	if got := get(t, store, bob, clip); got != (playbackutil.State{}) {
		t.Errorf("bob sees %+v of alice's state, want none", got)
	}
	save(t, store, bob, 99_000, 100_000)
	setWatched(t, store, bob, false)
	if got := get(t, store, alice, clip); got != aliceState {
		t.Errorf("alice's state after bob's changes = %+v, want %+v", got, aliceState)
	}
}

func TestState_IsPerDevice(t *testing.T) {
	store := newStore(t)
	save(t, store, alice, 30_000, 100_000)
	result, err := playbackutil.Get(context.Background(), playbackutil.GetParams{
		Store: store, UserID: alice, DeviceSerial: "USB1", RelPath: clip,
	})
	if err != nil || result.State != (playbackutil.State{}) {
		t.Errorf("same path on another device = %+v, %v, want the zero state", result.State, err)
	}
}
