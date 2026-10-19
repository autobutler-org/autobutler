package v0_music

import (
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/musicutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// rescanLibrary godoc
// @Summary Rescan the music library
// @Description Queues a full scan of every device, which reads each changed audio file and drops the tracks whose files are gone. Its progress shows as a music_rescan job. Admin only.
// @Tags music
// @Produce json
// @Success 202 {object} jobutil.Job
// @Failure 403 {object} serverutil.Response "Forbidden"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /music/rescan [post]
func rescanLibrary(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	principal, _ := ctxutil.Get[accessutil.Principal](c, "principal")
	result, err := musicutil.Rescan(musicutil.RescanParams{
		Ctx:    c.Request.Context(),
		Jobs:   deps.JobQueue(),
		UserID: principal.UserID,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Accepted().WithContentType(serverutil.ContentTypeJSON).WithData(result.Job)
}

var rescanLibraryRoute = serverutil.ApiRoute("POST", "/music/rescan", rescanLibrary)
