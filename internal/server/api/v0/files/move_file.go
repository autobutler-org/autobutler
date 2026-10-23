package v0_files

import (
	"path"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// moveFile godoc
// @Summary Move or rename a file
// @Description Enqueue a file move operation between paths/devices. Needs write access on the file and on the folder it moves into. A file already at the new path is replaced; a .qslide, .qsheet or .qdoc on the internal device is first snapshotted into its version history.
// @Tags files
// @Accept json
// @Produce json
// @Param body body moveFileRequest true "Move file request"
// @Success 202 {object} serverutil.Response "Ok"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 403 {object} serverutil.Response "Forbidden"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /files [put]
func moveFile(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	var req moveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}
	access, resp := loadAccess(c, deps)
	if resp != nil {
		return resp
	}
	// Moving things into a home or group folder is fine; moving the folder
	// itself is not.
	if refused := refuseHomeRoot(access, req.OldDeviceSerial, req.OldFilePath); refused != nil {
		return refused
	}
	for _, target := range []struct{ serial, path string }{
		{req.OldDeviceSerial, req.OldFilePath},
		{req.NewDeviceSerial, path.Dir(req.NewFilePath)},
	} {
		if resp := gate(access, target.serial, target.path, accessutil.Write); resp != nil {
			return resp
		}
	}

	if _, err := fileutil.MoveFile(fileutil.MoveFileParams{
		Ctx:             c.Request.Context(),
		Registry:        deps.VFSRegistry(),
		EventBus:        deps.EventBus(),
		Database:        deps.Database(),
		OldFilePath:     req.OldFilePath,
		NewFilePath:     req.NewFilePath,
		OldDeviceSerial: req.OldDeviceSerial,
		NewDeviceSerial: req.NewDeviceSerial,
		// A move that lands on an editor document snapshots it first (#2992).
		BeforeOverwrite: beforeSave(c, deps, req.NewDeviceSerial),
	}); err != nil {
		return fileError(err)
	}
	return serverutil.Ok()
}

var moveFileRoute = serverutil.ApiRoute(
	"PUT", "/files", moveFile,
)
