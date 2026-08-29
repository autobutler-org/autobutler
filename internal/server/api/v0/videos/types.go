package v0_videos

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getMetadataRoute,
		extractFrameRoute,
		trimVideoRoute,
	}
}

// extractFrameRequest is the POST body for /videos/extract-frame.
type extractFrameRequest struct {
	RelPath     string `json:"relPath"`
	Serial      string `json:"serial"`
	TimestampMs int64  `json:"timestampMs"`
}

// extractFrameResponse is returned on success.
type extractFrameResponse struct {
	RelPath string `json:"relPath"`
}

// trimVideoRequest is the POST body for /videos/trim.
type trimVideoRequest struct {
	RelPath string `json:"relPath"`
	Serial  string `json:"serial"`
	StartMs int64  `json:"startMs"`
	EndMs   int64  `json:"endMs"`
}

// trimVideoResponse is returned on success.
type trimVideoResponse struct {
	RelPath string `json:"relPath"`
}

type albumRefJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
