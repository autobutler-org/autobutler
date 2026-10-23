package v0_photos

import (
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// getIndexStatus godoc
// @Summary Get the photo library index status
// @Description Admin only. Reports the background pass that hashes and dates every library photo on this device: how many photos the library holds, how many have both hashes (which duplicate detection needs), how many have had their capture date read, how many failed and the first 500 of those with a reason (undecodable, missing_converter, too_large or unreadable), when a pass last finished, and whether one is running or paused. The state is in memory: a restart forgets a pause and starts a fresh pass that skips what is already indexed. photo_index_changed events say when to fetch it again.
// @Tags photos
// @Produce json
// @Success 200 {object} photoutil.IndexStatusResult
// @Failure 403 {object} serverutil.Response "Not an admin"
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /photos/index-status [get]
func getIndexStatus(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	result, err := photoutil.IndexStatus(photoutil.IndexStatusParams{Control: deps.PhotoIndex()})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

var getIndexStatusRoute = serverutil.ApiRoute("GET", "/photos/index-status", getIndexStatus)
