package v0_playback

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/playbackutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// setWatched godoc
// @Summary Mark a video watched or unwatched for the caller
// @Description Marking a video watched resets the caller's resume position to 0 and keeps the stored duration. Marking it unwatched forgets the caller's state for it, resume position included, and returns the zero state. Needs read access on the video.
// @Tags playback
// @Accept json
// @Produce json
// @Param body body setWatchedRequest true "Video reference and the watched flag"
// @Success 200 {object} playbackutil.State
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /playback/watched [put]
func setWatched(c *gin.Context) *serverutil.Response {
	var req setWatchedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("invalid request body"))
	}
	t, refusal := resolveTarget(c, req.DeviceSerial, req.RelPath)
	if refusal != nil {
		return refusal
	}
	result, err := playbackutil.SetWatched(c.Request.Context(), playbackutil.SetWatchedParams{
		Store:        t.store,
		UserID:       t.userID,
		DeviceSerial: t.deviceSerial,
		RelPath:      t.relPath,
		Watched:      req.Watched,
	})
	return stateResponse(result.State, err)
}

var setWatchedRoute = serverutil.ApiRoute("PUT", "/playback/watched", setWatched)
