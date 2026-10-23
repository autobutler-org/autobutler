package v0_photos

import (
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// pauseIndex godoc
// @Summary Pause the photo library index pass
// @Description Admin only. Holds the background pass that hashes and dates library photos before its next photo, leaving every IO slot to requests; the photo being read finishes first. A pass that has not started holds before its first photo. Pausing a paused pass changes nothing. The pause lasts until resumed or the server restarts, and publishes photo_index_changed.
// @Tags photos
// @Produce json
// @Success 200 {object} photoutil.IndexStatusResult
// @Failure 403 {object} serverutil.Response "Not an admin"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /photos/index/pause [post]
func pauseIndex(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	result, err := photoutil.PauseIndex(photoutil.PauseIndexParams{
		Control: deps.PhotoIndex(), EventBus: deps.EventBus(),
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

var pauseIndexRoute = serverutil.ApiRoute("POST", "/photos/index/pause", pauseIndex)
