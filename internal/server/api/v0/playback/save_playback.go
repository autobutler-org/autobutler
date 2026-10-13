package v0_playback

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/playbackutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// savePlayback godoc
// @Summary Save the caller's resume position for a video
// @Description Stores where the caller's player has got to. A position at or past 95% of the duration marks the video watched and resets the position to 0; an earlier one keeps the stored watched flag. Needs read access on the video.
// @Tags playback
// @Accept json
// @Produce json
// @Param body body savePlaybackRequest true "Video reference, position and duration in milliseconds"
// @Success 200 {object} playbackutil.State
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /playback [put]
func savePlayback(c *gin.Context) *serverutil.Response {
	var req savePlaybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("invalid request body"))
	}
	t, refusal := resolveTarget(c, req.DeviceSerial, req.RelPath)
	if refusal != nil {
		return refusal
	}
	result, err := playbackutil.Save(c.Request.Context(), playbackutil.SaveParams{
		Store:        t.store,
		UserID:       t.userID,
		DeviceSerial: t.deviceSerial,
		RelPath:      t.relPath,
		PositionMs:   req.PositionMs,
		DurationMs:   req.DurationMs,
	})
	return stateResponse(result.State, err)
}

var savePlaybackRoute = serverutil.ApiRoute("PUT", "/playback", savePlayback)
