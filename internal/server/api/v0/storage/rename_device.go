package v0_storage

import (
	"strings"
	"unicode"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

const maxDeviceNameLength = 64

// renameDevice godoc
// @Summary Rename a storage device
// @Description Sets a custom display name for a storage device identified by its serial number
// @Tags storage
// @Accept json
// @Produce json
// @Param serial query string true "Device serial (empty string for internal device)"
// @Param body body object true "{name: string}"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /storage/devices/rename [patch]
func renameDevice(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	serial := c.Query("serial")

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return serverutil.BadRequest(nil)
	}
	if len([]rune(name)) > maxDeviceNameLength {
		return serverutil.BadRequest(nil)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return serverutil.BadRequest(nil)
		}
	}

	if err := deps.Database().Queries.UpsertDeviceName(c.Request.Context(), db.UpsertDeviceNameParams{
		DeviceSerial: serial,
		DisplayName:  name,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(gin.H{"serial": serial, "name": name})
}

var renameDeviceRoute = serverutil.ApiRoute(
	"PATCH", "/storage/devices/rename", renameDevice,
)
