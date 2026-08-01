package v0_share

import (
	"errors"
	"fmt"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

type createShareRequest struct {
	// ResourceType is "file" or "folder".
	ResourceType string `json:"resourceType" binding:"required"`
	// ResourcePath is the file/folder path relative to the device's Cirrus root.
	ResourcePath string `json:"resourcePath" binding:"required"`
	// DeviceSerial identifies the storage device (empty for the default device).
	DeviceSerial string `json:"deviceSerial"`
	// ExpiresInHours is the TTL in hours. 0 or negative means no expiry.
	ExpiresInHours int `json:"expiresInHours"`
}

type createShareResponse struct {
	Token     string  `json:"token"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

// createShare godoc
// @Summary Create a share link
// @Description Generate an expiring public share link for a file or folder.
// @Tags share
// @Accept json
// @Produce json
// @Param body body createShareRequest true "Share parameters"
// @Success 201 {object} createShareResponse
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /share [post]
func createShare(c *gin.Context) *serverutil.Response {
	var req createShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("invalid request body"))
	}
	if req.ResourceType != "file" && req.ResourceType != "folder" {
		return serverutil.BadRequest(errors.New("resourceType must be \"file\" or \"folder\""))
	}

	username, ok := ctxutil.Get[string](c, "username")
	if !ok || username == "" {
		return serverutil.Unauthorized(errors.New("authentication required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(errors.New("database unavailable"))
	}

	// Resolve user ID from username.
	user, err := database.Queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("resolve user: %w", err))
	}

	var expiresAt *time.Time
	var expiresAtStr *string
	if req.ExpiresInHours > 0 {
		t := time.Now().UTC().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expiresAt = &t
		s := t.Format(time.RFC3339)
		expiresAtStr = &s
	}

	rawToken, err := authutil.CreateShareLink(c.Request.Context(), database.Queries, authutil.ShareLinkParams{
		UserID:       user.ID,
		ResourceType: req.ResourceType,
		ResourcePath: req.ResourcePath,
		DeviceSerial: req.DeviceSerial,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("create share link: %w", err))
	}

	return serverutil.Created(createShareResponse{
		Token:     rawToken,
		ExpiresAt: expiresAtStr,
	})
}

var createShareRoute = serverutil.ApiRoute("POST", "/share", createShare)
