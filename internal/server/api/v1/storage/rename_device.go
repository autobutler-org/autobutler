package v1_storage

import (
	"strings"
	"unicode"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

const maxDeviceNameLength = 64

// renameDevice godoc
// @Summary Rename a storage device
// @Description Sets a custom display name for a storage device identified by its device path
// @Tags storage
// @Accept json
// @Produce json
// @Param devicePath path string true "Device path (e.g. /dev/disk3s5 — leading slash included in wildcard)"
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

	// Device paths contain slashes (e.g. /dev/disk3s5) which can't be
	// embedded in a URL path segment without ambiguity. Accept via query param.
	devicePath := c.Query("devicePath")
	if devicePath == "" {
		return serverutil.BadRequest(nil)
	}

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
		DevicePath:  devicePath,
		DisplayName: name,
	}); err != nil {
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(gin.H{"devicePath": devicePath, "name": name})
}

var renameDeviceRoute = serverutil.ApiRoute(
	"PATCH", "/storage/devices/rename", renameDevice,
)
