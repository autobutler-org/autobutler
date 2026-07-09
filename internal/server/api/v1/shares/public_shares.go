package v1_shares

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
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/shareutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// sharePasswordHeader carries the share password on public requests. A query
// parameter (?password=) is also accepted for clients that cannot set headers,
// e.g. a plain browser link.
const sharePasswordHeader = "X-Share-Password"

type shareInfoJSON struct {
	// Name and IsFolder are omitted until the password (if any) is supplied,
	// so a protected link leaks nothing about its target.
	Name              string `json:"name,omitempty"`
	IsFolder          bool   `json:"isFolder"`
	SizeBytes         int64  `json:"sizeBytes,omitempty"`
	PasswordProtected bool   `json:"passwordProtected"`
	ExpiresAt         string `json:"expiresAt,omitempty"`
}

// suppliedPassword extracts the share password from header or query.
func suppliedPassword(c *gin.Context) string {
	if p := c.GetHeader(sharePasswordHeader); p != "" {
		return p
	}
	return c.Query("password")
}

// shareErrorResponse maps shareutil sentinel errors onto HTTP semantics.
// Unknown and expired shares are indistinguishable content-wise, but expired
// links return 410 so recipients see "this link expired" rather than a 404.
func shareErrorResponse(err error) *serverutil.Response {
	switch {
	case errors.Is(err, shareutil.ErrNotFound):
		return serverutil.NotFound(errors.New("share not found"))
	case errors.Is(err, shareutil.ErrExpired):
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusGone).
			WithError(errors.New("share link has expired"))
	case errors.Is(err, shareutil.ErrPasswordRequired):
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusUnauthorized).
			WithError(errors.New("password required"))
	case errors.Is(err, shareutil.ErrWrongPassword):
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusForbidden).
			WithError(errors.New("wrong password"))
	default:
		return serverutil.InternalServerError(err)
	}
}

// publicShareInfo godoc
// @Summary Get share link metadata (public)
// @Description Returns metadata about a share link. For password-protected shares the target name and size are withheld until the correct password is supplied via X-Share-Password or ?password=.
// @Tags shares
// @Produce json
// @Param token path string true "Share token"
// @Param password query string false "Share password"
// @Success 200 {object} shareInfoJSON
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 410 {object} serverutil.Response "Gone (expired)"
// @Router /public/shares/{token}/info [get]
func publicShareInfo(c *gin.Context) *serverutil.Response {
	share, err := shareutil.Peek(c.Param("token"))
	if err != nil {
		return shareErrorResponse(err)
	}

	info := shareInfoJSON{PasswordProtected: share.PasswordProtected()}
	if share.ExpiresAt != nil {
		info.ExpiresAt = share.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}

	// Withhold target details until the password checks out.
	if share.PasswordProtected() && !authutil.CheckPassword(suppliedPassword(c), share.PasswordHash) {
		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(info)
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	stat, err := deps.StorageService().StatFile(storageutil.StatFileParams{
		FilePath:     share.FilePath,
		DeviceSerial: share.DeviceSerial,
	})
	if err != nil {
		// The underlying file was deleted or its device unplugged.
		return serverutil.NotFound(errors.New("shared file is no longer available"))
	}
	info.Name = stat.Name
	info.IsFolder = stat.IsDir
	if !stat.IsDir {
		if fi, err := os.Stat(stat.FullPath); err == nil {
			info.SizeBytes = fi.Size()
		}
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(info)
}

// publicShareDownload godoc
// @Summary Download a shared file or folder (public)
// @Description Streams the shared file (folders are zipped). Always served as an attachment with sniffing disabled so shared content can never execute in the app's origin.
// @Tags shares
// @Produce application/octet-stream
// @Param token path string true "Share token"
// @Param password query string false "Share password"
// @Success 200 {file} file
// @Failure 401 {object} serverutil.Response "Password required"
// @Failure 403 {object} serverutil.Response "Wrong password"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 410 {object} serverutil.Response "Gone (expired)"
// @Router /public/shares/{token} [get]
func publicShareDownload(c *gin.Context) *serverutil.Response {
	share, err := shareutil.Resolve(c.Param("token"), suppliedPassword(c))
	if err != nil {
		return shareErrorResponse(err)
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	result, err := deps.StorageService().DownloadFile(storageutil.DownloadFileParams{
		FilePath:     share.FilePath,
		DeviceSerial: share.DeviceSerial,
	})
	if err != nil {
		return serverutil.NotFound(errors.New("shared file is no longer available"))
	}

	// Defense in depth: this endpoint is unauthenticated and serves
	// user-chosen files on the app's origin, so never let the browser render
	// or sniff the content — a shared .html file must not become stored XSS.
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox")
	c.Header("Content-Type", "application/octet-stream")

	if result.IsFolder {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(result.FullPath)+".zip"))
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		if err := zipWriter.AddFS(os.DirFS(result.FullPath)); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to zip shared folder: %w", err))
		}
		return nil // response written directly to writer
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(result.FullPath)))
	c.File(result.FullPath)
	return nil // response written directly via c.File
}

var publicShareInfoRoute = serverutil.ApiRoute("GET", "/public/shares/:token/info", publicShareInfo)
var publicShareDownloadRoute = serverutil.ApiRoute("GET", "/public/shares/:token", publicShareDownload)
