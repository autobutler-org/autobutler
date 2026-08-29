package v0_photos

import (
	"context"
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// rotatePhoto godoc
// @Summary Save the rotation for a photo
// @Description Persists the viewer rotation (0/1/2/3 × 90° CW) for a photo server-side.
// @Tags photos
// @Accept json
// @Produce json
// @Param body body rotatePhotoRequest true "Rotation request"
// @Success 200 {object} serverutil.Response
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos/rotate [post]
func rotatePhoto(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	var req rotatePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
	}

	if err := photoutil.SaveRotation(photoutil.SaveRotationParams{
		Ctx:              context.Background(),
		Queries:          deps.Database().Queries,
		Serial:           req.Serial,
		RelPath:          req.RelPath,
		RotationQuarters: req.RotationQuarters,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok()
}

var rotatePhotoRoute = serverutil.ApiRoute("POST", "/photos/rotate", rotatePhoto)
