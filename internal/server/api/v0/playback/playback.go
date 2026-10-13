// Package v0_playback serves /api/v0/playback: reading and saving how far the
// caller has watched a video, and marking one watched or unwatched. The state
// belongs to the caller's account; no route is admin-only.
package v0_playback

import "github.com/autobutler-org/quark/pkg/util/serverutil"

func NewRouter() serverutil.Router {
	return &router{}
}
