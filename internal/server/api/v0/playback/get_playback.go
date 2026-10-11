package v0_playback

import (
	"github.com/autobutler-org/quark/pkg/util/playbackutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// getPlayback godoc
// @Summary Get the caller's playback state for a video
// @Description Returns the caller's own resume position and watched flag for the video, all zero with no updatedAt when nothing is stored. Needs read access on the video.
// @Tags playback
// @Produce json
// @Param relPath query string true "Files-relative path of the video"
// @Param deviceSerial query string false "Device serial number; empty for the internal drive"
// @Success 200 {object} playbackutil.State
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /playback [get]
func getPlayback(c *gin.Context) *serverutil.Response {
	t, refusal := resolveTarget(c, c.Query("deviceSerial"), c.Query("relPath"))
	if refusal != nil {
		return refusal
	}
	result, err := playbackutil.Get(c.Request.Context(), playbackutil.GetParams{
		Store:        t.store,
		UserID:       t.userID,
		DeviceSerial: t.deviceSerial,
		RelPath:      t.relPath,
	})
	return stateResponse(result.State, err)
}

var getPlaybackRoute = serverutil.ApiRoute("GET", "/playback", getPlayback)
