package v1_storage

import (
	"fmt"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var validRoles = map[string]bool{
	"default-storage": true,
	"snapshot-backup": true,
	"unassigned":      true,
}

// setDeviceRole godoc
// @Summary Set the role of a storage device
// @Description Assigns a role (default-storage, snapshot-backup, unassigned) to a device. Requires master password re-entry.
// @Tags storage
// @Accept json
// @Produce json
// @Param body body object true "{serial: string, role: string, username: string, password: string}"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /storage/devices/role [put]
func setDeviceRole(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		Serial   string `json:"serial"`
		Role     string `json:"role" binding:"required"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	if !validRoles[req.Role] {
		return serverutil.BadRequest(fmt.Errorf("role must be one of: default-storage, snapshot-backup, unassigned"))
	}

	ctx := c.Request.Context()

	// Verify username matches the authenticated session.
	if sessionUser, ok := ctxutil.Get[string](c, "username"); ok && sessionUser != req.Username {
		return serverutil.Unauthorized(fmt.Errorf("username does not match session"))
	}

	// Re-validate master password.
	if _, err := authutil.ValidateBasicAuth(ctx, deps.Database().Queries, req.Username, req.Password); err != nil {
		return serverutil.Unauthorized(fmt.Errorf("invalid credentials"))
	}

	// For non-unassigned roles, verify the device is actually connected.
	if req.Role != "unassigned" {
		statuses, err := deps.StorageService().GetDeviceStatuses()
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to list devices: %w", err))
		}
		found := false
		isInternal := false
		for _, s := range statuses {
			if req.Serial == "" && s.IsInternal {
				found = true
				isInternal = true
				break
			}
			if s.UsbInfo != nil && s.UsbInfo.GetSerial() == req.Serial {
				found = true
				break
			}
		}
		if !found {
			return serverutil.BadRequest(fmt.Errorf("device with serial %q not found", req.Serial))
		}
		if isInternal && req.Role == "snapshot-backup" {
			return serverutil.BadRequest(fmt.Errorf("internal device cannot be assigned as snapshot-backup"))
		}
	}

	// If assigning default-storage, clear any existing default-storage first.
	if req.Role == "default-storage" {
		if err := deps.Database().Queries.ClearDefaultStorageRole(ctx); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to clear existing default-storage: %w", err))
		}
	}

	if err := deps.Database().Queries.UpsertDeviceRole(ctx, db.UpsertDeviceRoleParams{
		DeviceSerial: req.Serial,
		Role:         req.Role,
	}); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to set device role: %w", err))
	}

	return serverutil.Ok().WithData(gin.H{
		"serial": req.Serial,
		"role":   req.Role,
	})
}

var setDeviceRoleRoute = serverutil.ApiRoute(
	"PUT", "/storage/devices/role", setDeviceRole,
)
