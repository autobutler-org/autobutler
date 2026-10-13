package playbackutil

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/autobutler-org/quark/pkg/vfs"
)

// metadataKey is the per-account key a video's state is stored under, which
// is what keeps one account's state out of another's.
func metadataKey(userID int64) string {
	return "playback." + strconv.FormatInt(userID, 10)
}

// cleanRelPath spells a path one way, slash-separated with no leading slash,
// so "/a.mp4" and "a.mp4" share a row and ".." cannot leave the files root.
func cleanRelPath(p string) string {
	return strings.TrimPrefix(path.Clean("/"+p), "/")
}

func load(ctx context.Context, store vfs.MetadataStore, userID int64, serial, relPath string) (State, error) {
	meta, err := store.Get(ctx, vfs.FilesNamespace(serial), cleanRelPath(relPath))
	if err != nil {
		return State{}, err
	}
	raw, ok := meta[metadataKey(userID)]
	if !ok {
		return State{}, nil
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, fmt.Errorf("decode playback state: %w", err)
	}
	return state, nil
}

// save stamps state with the current time, writes it, and returns it as
// written.
func save(ctx context.Context, store vfs.MetadataStore, userID int64, serial, relPath string, state State) (State, error) {
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.Marshal(state)
	if err != nil {
		return State{}, err
	}
	err = store.Set(ctx, vfs.FilesNamespace(serial), cleanRelPath(relPath), map[string]json.RawMessage{metadataKey(userID): raw})
	if err != nil {
		return State{}, err
	}
	return state, nil
}
