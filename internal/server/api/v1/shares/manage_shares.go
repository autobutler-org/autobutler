package v1_shares

import (
	"errors"
	"fmt"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/shareutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// publicSharePathPrefix is where public share links are served. Kept in one
// place so the URL returned to clients and the routes can't drift apart.
const publicSharePathPrefix = "/api/v1/public/shares/"

type createShareRequest struct {
	FilePath     string `json:"filePath"`
	DeviceSerial string `json:"deviceSerial"`
	// Password, when non-empty, protects the link.
	Password string `json:"password"`
	// ExpiresInHours, when > 0, sets the link expiry relative to now.
	// 0 means the link never expires.
	ExpiresInHours int `json:"expiresInHours"`
}

type shareJSON struct {
	ID                string     `json:"id"`
	FilePath          string     `json:"filePath"`
	DeviceSerial      string     `json:"deviceSerial,omitempty"`
	URLPath           string     `json:"urlPath"`
	PasswordProtected bool       `json:"passwordProtected"`
	CreatedAt         time.Time  `json:"createdAt"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
	Expired           bool       `json:"expired"`
	AccessCount       int64      `json:"accessCount"`
	LastAccessAt      *time.Time `json:"lastAccessAt,omitempty"`
}

// toShareJSON maps a share record to its API shape. The bcrypt hash never
// leaves the server; the token only appears embedded in urlPath.
func toShareJSON(s shareutil.Share) shareJSON {
	return shareJSON{
		ID:                s.ID,
		FilePath:          s.FilePath,
		DeviceSerial:      s.DeviceSerial,
		URLPath:           publicSharePathPrefix + s.Token,
		PasswordProtected: s.PasswordProtected(),
		CreatedAt:         s.CreatedAt,
		ExpiresAt:         s.ExpiresAt,
		Expired:           s.IsExpired(),
		AccessCount:       s.AccessCount,
		LastAccessAt:      s.LastAccessAt,
	}
}

// createShare godoc
// @Summary Create a public share link
// @Description Creates a tokenized public link for a file or folder, optionally password-protected and/or expiring.
// @Tags shares
// @Accept json
// @Produce json
// @Param body body createShareRequest true "Share definition"
// @Success 200 {object} shareJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /shares [post]
func createShare(c *gin.Context) *serverutil.Response {
	var req createShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("invalid request body"))
	}
	if req.FilePath == "" {
		return serverutil.BadRequest(errors.New("filePath is required"))
	}
	if req.ExpiresInHours < 0 {
		return serverutil.BadRequest(errors.New("expiresInHours must not be negative"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	// Only existing paths can be shared — this also runs the safeJoin
	// traversal guard before anything is persisted.
	if _, err := deps.StorageService().StatFile(storageutil.StatFileParams{
		FilePath:     req.FilePath,
		DeviceSerial: req.DeviceSerial,
	}); err != nil {
		return serverutil.NotFound(fmt.Errorf("cannot share: %w", err))
	}

	var expiresAt *time.Time
	if req.ExpiresInHours > 0 {
		t := time.Now().UTC().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expiresAt = &t
	}

	result, err := shareutil.Create(shareutil.CreateShareParams{
		FilePath:     req.FilePath,
		DeviceSerial: req.DeviceSerial,
		Password:     req.Password,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(toShareJSON(result.Share))
}

// listShares godoc
// @Summary List share links
// @Description Returns all share links, newest first, including expired ones.
// @Tags shares
// @Produce json
// @Success 200 {array} shareJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /shares [get]
func listShares(c *gin.Context) *serverutil.Response {
	shares, err := shareutil.List()
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	out := make([]shareJSON, 0, len(shares))
	for _, s := range shares {
		out = append(out, toShareJSON(s))
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(out)
}

// deleteShare godoc
// @Summary Revoke a share link
// @Description Deletes a share link by ID. The public URL stops working immediately.
// @Tags shares
// @Produce json
// @Success 200 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /shares/{id} [delete]
func deleteShare(c *gin.Context) *serverutil.Response {
	id := c.Param("id")
	if err := shareutil.Delete(id); err != nil {
		if errors.Is(err, shareutil.ErrNotFound) {
			return serverutil.NotFound(err)
		}
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{"deleted": id})
}

var createShareRoute = serverutil.ApiRoute("POST", "/shares", createShare)
var listSharesRoute = serverutil.ApiRoute("GET", "/shares", listShares)
var deleteShareRoute = serverutil.ApiRoute("DELETE", "/shares/:id", deleteShare)
