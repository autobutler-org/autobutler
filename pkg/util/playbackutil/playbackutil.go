// Package playbackutil remembers how far each account has watched each video:
// a resume position and a watched flag, one row per (user, device, path) in
// the VFS metadata store under the key "playback.<user id>". Nothing here
// reads the video itself; the player reports positions and durations.
//
// State is keyed by path, so moving or renaming a video drops it: nothing
// rewrites vfs_metadata on a move today.
package playbackutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/pkg/vfs"
)

// WatchedThresholdPercent is how far into a video a saved position has to be,
// as a percentage of its duration, for the video to count as watched.
const WatchedThresholdPercent = 95

// ErrInvalid reports a position or duration that cannot be stored: a negative
// one.
var ErrInvalid = errors.New("invalid playback state")

// State is one account's playback state for one video. The zero State is what
// a video nobody has started has.
type State struct {
	// PositionMs is where to resume, in milliseconds. It is 0 once the video
	// counts as watched, so watching it again starts from the beginning.
	PositionMs int64 `json:"positionMs"`
	// DurationMs is the video's length as the player last reported it.
	DurationMs int64 `json:"durationMs"`
	// Watched stays set until SetWatched clears it.
	Watched bool `json:"watched"`
	// UpdatedAt is when the state was stored, RFC 3339 in UTC. Empty when
	// nothing is stored.
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// GetParams names the account and the video whose state to read.
type GetParams struct {
	Store        vfs.MetadataStore
	UserID       int64
	DeviceSerial string
	RelPath      string
}

// GetResult is the stored state, zero when there is none.
type GetResult struct {
	State State
}

// SaveParams is where an account's player has got to in a video.
type SaveParams struct {
	Store        vfs.MetadataStore
	UserID       int64
	DeviceSerial string
	RelPath      string
	PositionMs   int64
	DurationMs   int64
}

// SaveResult is the state as stored.
type SaveResult struct {
	State State
}

// SetWatchedParams marks a video watched or unwatched for an account.
type SetWatchedParams struct {
	Store        vfs.MetadataStore
	UserID       int64
	DeviceSerial string
	RelPath      string
	Watched      bool
}

// SetWatchedResult is the state as stored, zero after marking it unwatched.
type SetWatchedResult struct {
	State State
}

// Get returns an account's playback state for a video, the zero State when it
// has none.
func Get(ctx context.Context, params GetParams) (GetResult, error) {
	state, err := load(ctx, params.Store, params.UserID, params.DeviceSerial, params.RelPath)
	return GetResult{State: state}, err
}

// Save stores a resume position. A position at or past
// WatchedThresholdPercent of the duration marks the video watched and resets
// the position to 0; an earlier one keeps whatever watched flag is stored, so
// watching a watched video again leaves it watched. A negative position or
// duration is ErrInvalid.
func Save(ctx context.Context, params SaveParams) (SaveResult, error) {
	if params.PositionMs < 0 || params.DurationMs < 0 {
		return SaveResult{}, fmt.Errorf("%w: positionMs and durationMs must not be negative", ErrInvalid)
	}
	// ponytail: read-then-write without a transaction, so two racing saves from
	// one account can lose the sticky watched flag; move to a single upsert if
	// that ever shows up.
	previous, err := load(ctx, params.Store, params.UserID, params.DeviceSerial, params.RelPath)
	if err != nil {
		return SaveResult{}, err
	}
	state := State{PositionMs: params.PositionMs, DurationMs: params.DurationMs, Watched: previous.Watched}
	if params.DurationMs > 0 && params.PositionMs >= params.DurationMs*WatchedThresholdPercent/100 {
		state.Watched = true
		state.PositionMs = 0
	}
	state, err = save(ctx, params.Store, params.UserID, params.DeviceSerial, params.RelPath, state)
	return SaveResult{State: state}, err
}

// SetWatched marks a video watched, resetting its resume position and keeping
// its stored duration, or unwatched, which forgets the state entirely.
func SetWatched(ctx context.Context, params SetWatchedParams) (SetWatchedResult, error) {
	namespace, relPath := vfs.FilesNamespace(params.DeviceSerial), cleanRelPath(params.RelPath)
	if !params.Watched {
		err := params.Store.DeleteKeys(ctx, namespace, relPath, []string{metadataKey(params.UserID)})
		return SetWatchedResult{}, err
	}
	previous, err := load(ctx, params.Store, params.UserID, params.DeviceSerial, params.RelPath)
	if err != nil {
		return SetWatchedResult{}, err
	}
	state, err := save(ctx, params.Store, params.UserID, params.DeviceSerial, params.RelPath,
		State{DurationMs: previous.DurationMs, Watched: true})
	return SetWatchedResult{State: state}, err
}
