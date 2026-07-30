package v0_devices

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// RegisteredDeviceJSON is the API representation of a registered device.
type RegisteredDeviceJSON struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	DeviceType     string     `json:"deviceType"`
	IdentityType   string     `json:"identityType"`
	IPAddress      string     `json:"ipAddress"`
	MACAddress     *string    `json:"macAddress,omitempty"`
	TailscaleKey   *string    `json:"tailscaleKey,omitempty"`
	UserAgent      string     `json:"userAgent"`
	ApprovalStatus string     `json:"approvalStatus"`
	ApprovedBy     *string    `json:"approvedBy,omitempty"`
	ApprovedAt     *time.Time `json:"approvedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func toRegisteredDeviceJSON(d db.RegisteredDevice) RegisteredDeviceJSON {
	j := RegisteredDeviceJSON{
		ID:             d.ID,
		Name:           d.Name,
		DeviceType:     d.DeviceType,
		IdentityType:   d.IdentityType,
		IPAddress:      d.IpAddress,
		UserAgent:      d.UserAgent,
		ApprovalStatus: d.ApprovalStatus,
		CreatedAt:      d.CreatedAt,
	}
	if d.MacAddress.Valid {
		j.MACAddress = &d.MacAddress.String
	}
	if d.TailscaleKey.Valid {
		j.TailscaleKey = &d.TailscaleKey.String
	}
	if d.ApprovedBy.Valid {
		j.ApprovedBy = &d.ApprovedBy.String
	}
	if d.ApprovedAt.Valid {
		j.ApprovedAt = &d.ApprovedAt.Time
	}
	return j
}

// registerDevice godoc
// @Summary Register a device
// @Description Submits a device for admin approval. If the device is already registered, updates its metadata and returns its current status. The endpoint is public so unapproved devices can call it.
// @Tags devices
// @Accept json
// @Produce json
// @Param body body object true "{name, deviceType, macAddress}"
// @Success 200 {object} RegisteredDeviceJSON
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /devices/register [post]
var registerDeviceRoute = serverutil.ApiRoute(
	"POST", "/devices/register", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}

		var req struct {
			Name       string `json:"name"`
			DeviceType string `json:"deviceType"`
			MACAddress string `json:"macAddress"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(err)
		}

		ip := c.ClientIP()
		ua := c.Request.UserAgent()

		d, err := deps.Database().Queries.RegisterDevice(c.Request.Context(), db.RegisterDeviceParams{
			Name:         req.Name,
			DeviceType:   req.DeviceType,
			IdentityType: "local",
			IpAddress:    ip,
			MacAddress:   sql.NullString{String: req.MACAddress, Valid: req.MACAddress != ""},
			TailscaleKey: sql.NullString{},
			UserAgent:    ua,
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithData(toRegisteredDeviceJSON(d))
	},
)

// listRegisteredDevices godoc
// @Summary List registered devices
// @Description Returns all registered devices, optionally filtered by approval status (pending, approved, revoked).
// @Tags devices
// @Produce json
// @Param status query string false "Filter by status: pending | approved | revoked"
// @Success 200 {array} RegisteredDeviceJSON
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /devices/registered [get]
var listRegisteredDevicesRoute = serverutil.ApiRoute(
	"GET", "/devices/registered", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}

		ctx := c.Request.Context()
		status := c.Query("status")

		var rows []db.RegisteredDevice
		var err error
		if status != "" {
			rows, err = deps.Database().Queries.ListRegisteredDevicesByStatus(ctx, status)
		} else {
			rows, err = deps.Database().Queries.ListRegisteredDevices(ctx)
		}
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		result := make([]RegisteredDeviceJSON, len(rows))
		for i, d := range rows {
			result[i] = toRegisteredDeviceJSON(d)
		}
		return serverutil.Ok().WithData(result)
	},
)

// approveDevice godoc
// @Summary Approve a device
// @Description Grants a pending device access to the butler API.
// @Tags devices
// @Param id path int true "Registered device ID"
// @Produce json
// @Success 200 {object} RegisteredDeviceJSON
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /devices/registered/{id}/approve [post]
var approveDeviceRoute = serverutil.ApiRoute(
	"POST", "/devices/registered/:id/approve", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(err)
		}
		username, _ := ctxutil.Get[string](c, "username")
		d, err := deps.Database().Queries.ApproveDevice(c.Request.Context(), db.ApproveDeviceParams{
			ApprovedBy: sql.NullString{String: username, Valid: username != ""},
			ID:         id,
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(toRegisteredDeviceJSON(d))
	},
)

// revokeDevice godoc
// @Summary Revoke a device
// @Description Revokes an approved or pending device, preventing future API access.
// @Tags devices
// @Param id path int true "Registered device ID"
// @Produce json
// @Success 200 {object} RegisteredDeviceJSON
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /devices/registered/{id}/revoke [post]
var revokeDeviceRoute = serverutil.ApiRoute(
	"POST", "/devices/registered/:id/revoke", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(err)
		}
		username, _ := ctxutil.Get[string](c, "username")
		d, err := deps.Database().Queries.RevokeDevice(c.Request.Context(), db.RevokeDeviceParams{
			ApprovedBy: sql.NullString{String: username, Valid: username != ""},
			ID:         id,
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(toRegisteredDeviceJSON(d))
	},
)

// deleteRegisteredDevice godoc
// @Summary Delete a registered device record
// @Description Permanently removes a registered device entry.
// @Tags devices
// @Param id path int true "Registered device ID"
// @Success 200 {object} serverutil.Response
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /devices/registered/{id} [delete]
var deleteRegisteredDeviceRoute = serverutil.ApiRoute(
	"DELETE", "/devices/registered/:id", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(err)
		}
		if err := deps.Database().Queries.DeleteRegisteredDevice(c.Request.Context(), id); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok()
	},
)

// pendingDeviceCount godoc
// @Summary Count pending device registrations
// @Description Returns the number of devices awaiting admin approval. Useful for notification badges.
// @Tags devices
// @Produce json
// @Success 200 {object} object{count=int}
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /devices/registered/pending/count [get]
var pendingDeviceCountRoute = serverutil.ApiRoute(
	"GET", "/devices/registered/pending/count", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		count, err := deps.Database().Queries.CountPendingDevices(c.Request.Context())
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(gin.H{
			"count": count,
		})
	},
)

// getRegisteredDevice godoc
// @Summary Get a registered device
// @Description Returns a single registered device by ID.
// @Tags devices
// @Param id path int true "Registered device ID"
// @Produce json
// @Success 200 {object} RegisteredDeviceJSON
// @Failure 400 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /devices/registered/{id} [get]
var getRegisteredDeviceRoute = serverutil.ApiRoute(
	"GET", "/devices/registered/:id", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(err)
		}
		d, err := deps.Database().Queries.GetRegisteredDevice(c.Request.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return serverutil.NotFound(err)
			}
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(toRegisteredDeviceJSON(d))
	},
)

// checkDeviceAccess godoc
// @Summary Check device approval status
// @Description Returns whether the calling device is approved. Intended for unapproved devices to poll until they are approved.
// @Tags devices
// @Produce json
// @Success 200 {object} object{status=string,approved=bool}
// @Router /devices/access [get]
var checkDeviceAccessRoute = serverutil.ApiRoute(
	"GET", "/devices/access", func(c *gin.Context) *serverutil.Response {
		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(nil)
		}

		ip := c.ClientIP()
		ua := c.Request.UserAgent()

		d, err := deps.Database().Queries.GetRegisteredDeviceByIPUA(c.Request.Context(),
			db.GetRegisteredDeviceByIPUAParams{
				IpAddress: ip,
				UserAgent: ua,
			})
		if err != nil {
			// Not registered at all.
			return serverutil.Ok().WithData(gin.H{
				"status":   "unknown",
				"approved": false,
			})
		}

		return serverutil.Ok().WithData(gin.H{
			"status":   d.ApprovalStatus,
			"approved": d.ApprovalStatus == "approved",
		})
	},
)

// Ensure http is imported for StatusNotFound when called via serverutil.NotFound.
var _ = http.StatusNotFound
