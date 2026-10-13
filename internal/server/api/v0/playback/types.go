package v0_playback

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// errNoAccess is what a caller hears about a video they may not see. It reads
// the same as a video that does not exist, because to them it does not.
var errNoAccess = errors.New("video not found")

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getPlaybackRoute,
		savePlaybackRoute,
		setWatchedRoute,
	}
}

// target is the video a request names, once the caller is known to be able to
// read it and it is known to exist.
type target struct {
	store        vfs.MetadataStore
	userID       int64
	deviceSerial string
	relPath      string
}

type savePlaybackRequest struct {
	RelPath      string `json:"relPath"`
	DeviceSerial string `json:"deviceSerial"`
	PositionMs   int64  `json:"positionMs"`
	DurationMs   int64  `json:"durationMs"`
}

type setWatchedRequest struct {
	RelPath      string `json:"relPath"`
	DeviceSerial string `json:"deviceSerial"`
	Watched      bool   `json:"watched"`
}
