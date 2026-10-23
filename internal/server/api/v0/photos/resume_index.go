package v0_photos

import (
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// resumeIndex godoc
// @Summary Resume the photo library index pass
// @Description Admin only. Lets a paused photo library index pass carry on from the photo it stopped before. Resuming a pass that is not paused changes nothing. Publishes photo_index_changed.
// @Tags photos
// @Produce json
// @Success 200 {object} photoutil.IndexStatusResult
// @Failure 403 {object} serverutil.Response "Not an admin"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /photos/index/resume [post]
func resumeIndex(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	result, err := photoutil.ResumeIndex(photoutil.ResumeIndexParams{
		Control: deps.PhotoIndex(), EventBus: deps.EventBus(),
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

var resumeIndexRoute = serverutil.ApiRoute("POST", "/photos/index/resume", resumeIndex)
