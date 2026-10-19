package v0_music

import (
	"github.com/autobutler-org/quark/pkg/util/musicutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// getStatus godoc
// @Summary Get the music library's status
// @Description Returns how many tracks the caller can read, when the last full scan finished, and the scan pending or running now, so an empty Music page can say "scanning" rather than "no music".
// @Tags music
// @Produce json
// @Success 200 {object} musicutil.StatusResult
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /music/status [get]
func getStatus(c *gin.Context) *serverutil.Response {
	who, refusal := loadCaller(c)
	if refusal != nil {
		return refusal
	}
	result, err := musicutil.Status(musicutil.StatusParams{
		Ctx:    c.Request.Context(),
		DB:     who.deps.Database().Db,
		Access: who.access,
		Jobs:   who.deps.JobQueue(),
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return jsonOk(result)
}

var getStatusRoute = serverutil.ApiRoute("GET", "/music/status", getStatus)
