package v0_share

import (
	"archive/zip"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/ratelimitutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// publicShareRateLimiter throttles unauthenticated share-link access per IP:
// 10 req/s sustained, burst of 20.
var publicShareRateLimiter = ratelimitutil.NewWithRate(10, 20)

type publicShareMeta struct {
	ResourceType string `json:"resourceType"`
	ResourcePath string `json:"resourcePath"`
	DeviceSerial string `json:"deviceSerial"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	ViewCount    int64  `json:"viewCount"`
}

// getPublicShare godoc
// @Summary Get share link metadata
// @Description Returns metadata for a public share link (no auth required).
// @Tags share
// @Param token path string true "Raw share token"
// @Success 200 {object} publicShareMeta
// @Failure 404 {object} serverutil.Response "Share not found or expired"
// @Failure 429 {object} serverutil.Response "Rate limit exceeded"
// @Router /public/share/{token} [get]
func getPublicShare(c *gin.Context) *serverutil.Response {
	if !publicShareRateLimiter.Allow(c.ClientIP()) {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusTooManyRequests).
			WithError(errors.New("rate limit exceeded"))
	}

	rawToken := c.Param("token")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(errors.New("database unavailable"))
	}

	link, err := authutil.ValidateShareLink(c.Request.Context(), database.Queries, rawToken)
	if err != nil {
		return serverutil.NotFound(errors.New("share link not found or expired"))
	}

	// Increment view count asynchronously.
	go func() {
		_ = database.Queries.IncrementShareLinkViewCount(
			c.Request.Context(),
			link.TokenHash,
		)
	}()

	meta := publicShareMeta{
		ResourceType: link.ResourceType,
		ResourcePath: link.ResourcePath,
		DeviceSerial: link.DeviceSerial,
		ViewCount:    link.ViewCount,
	}
	if link.ExpiresAt.Valid {
		meta.ExpiresAt = link.ExpiresAt.Time.Format("2006-01-02T15:04:05Z")
	}

	return serverutil.Ok().WithData(meta)
}

// downloadPublicShare godoc
// @Summary Download shared content
// @Description Stream the file (or zip of folder) for a public share link.
// @Tags share
// @Param token path string true "Raw share token"
// @Success 200 {file} file
// @Failure 404 {object} serverutil.Response "Share not found, expired, or resource missing"
// @Failure 429 {object} serverutil.Response "Rate limit exceeded"
// @Router /public/share/{token}/download [get]
func downloadPublicShare(c *gin.Context) *serverutil.Response {
	if !publicShareRateLimiter.Allow(c.ClientIP()) {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusTooManyRequests).
			WithError(errors.New("rate limit exceeded"))
	}

	rawToken := c.Param("token")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(errors.New("database unavailable"))
	}

	link, err := authutil.ValidateShareLink(c.Request.Context(), database.Queries, rawToken)
	if err != nil {
		return serverutil.NotFound(errors.New("share link not found or expired"))
	}

	result, err := deps.StorageService().DownloadFile(storageutil.DownloadFileParams{
		FilePath:     link.ResourcePath,
		DeviceSerial: link.DeviceSerial,
	})
	if err != nil {
		return serverutil.NotFound(fmt.Errorf("resource not found: %w", err))
	}

	base := filepath.Base(result.FullPath)

	if result.IsFolder {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, base))
		c.Header("Content-Type", "application/zip")
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		dirFS := os.DirFS(result.FullPath)
		if err := zipWriter.AddFS(dirFS); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("zip folder: %w", err))
		}
		return nil // response already written
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, base))
	c.File(result.FullPath)
	return nil // response already written
}

var getPublicShareRoute = serverutil.ApiRoute("GET", "/public/share/:token", getPublicShare)
var downloadPublicShareRoute = serverutil.ApiRoute("GET", "/public/share/:token/download", downloadPublicShare)
