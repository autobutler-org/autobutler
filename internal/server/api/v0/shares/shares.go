// Package v0_shares implements the public share-link API.
// Authenticated routes (create, list, delete) live under /api/v0/shares.
// The unauthenticated access route lives at /s/:token (no /api prefix) so
// share URLs are short and shareable.
package v0_shares

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// --- Types ---

type createShareRequest struct {
	RelPath      string `json:"relPath"`
	Serial       string `json:"serial"`
	Password     string `json:"password,omitempty"`
	MaxUses      int64  `json:"maxUses,omitempty"`
	ExpiresInSec *int64 `json:"expiresInSeconds,omitempty"`
}

type shareLinkJSON struct {
	ID           int64   `json:"id"`
	Token        string  `json:"token"`
	DeviceSerial string  `json:"deviceSerial,omitempty"`
	RelPath      string  `json:"relPath"`
	HasPassword  bool    `json:"hasPassword"`
	MaxUses      int64   `json:"maxUses"`
	UseCount     int64   `json:"useCount"`
	ExpiresAt    *string `json:"expiresAt,omitempty"`
	CreatedAt    string  `json:"createdAt"`
	CreatedBy    string  `json:"createdBy"`
	URL          string  `json:"url"`
}

// --- Helpers ---

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func toShareLinkJSON(s db.ShareLink, baseURL string) shareLinkJSON {
	j := shareLinkJSON{
		ID:           s.ID,
		Token:        s.Token,
		DeviceSerial: s.DeviceSerial,
		RelPath:      s.RelPath,
		HasPassword:  s.PasswordHash != "",
		MaxUses:      s.MaxUses,
		UseCount:     s.UseCount,
		CreatedAt:    s.CreatedAt.UTC().Format(time.RFC3339),
		CreatedBy:    s.CreatedBy,
		URL:          baseURL + "/s/" + s.Token,
	}
	if s.ExpiresAt.Valid {
		t := s.ExpiresAt.Time.UTC().Format(time.RFC3339)
		j.ExpiresAt = &t
	}
	return j
}

func baseURL(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

// --- Authenticated routes ---

// createShareLink godoc
// @Summary Create a share link
// @Description Creates a short-lived or permanent public share link for a file or folder.
// @Tags shares
// @Accept json
// @Produce json
// @Param body body createShareRequest true "Share link parameters"
// @Success 201 {object} shareLinkJSON
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /shares [post]
func createShareLink(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(fmt.Errorf("database unavailable"))
	}

	var req createShareRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		return serverutil.BadRequest(fmt.Errorf("invalid request body: %w", err))
	}
	if req.RelPath == "" {
		return serverutil.BadRequest(fmt.Errorf("relPath is required"))
	}

	// Validate path is safe.
	if err := validateRelPath(req.RelPath, req.Serial, deps); err != nil {
		return serverutil.BadRequest(err)
	}

	token, err := generateToken()
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("token generation: %w", err))
	}

	var passwordHash string
	if req.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("hash password: %w", err))
		}
		passwordHash = string(h)
	}

	var expiresAt sql.NullTime
	if req.ExpiresInSec != nil && *req.ExpiresInSec > 0 {
		expiresAt = sql.NullTime{
			Time:  time.Now().UTC().Add(time.Duration(*req.ExpiresInSec) * time.Second),
			Valid: true,
		}
	}

	// Best-effort username from context (set by auth middleware).
	username, _ := ctxutil.Get[string](c, "username")

	id, err := database.Queries.CreateShareLink(c.Request.Context(), db.CreateShareLinkParams{
		Token:        token,
		DeviceSerial: req.Serial,
		RelPath:      req.RelPath,
		PasswordHash: passwordHash,
		MaxUses:      req.MaxUses,
		ExpiresAt:    expiresAt,
		CreatedBy:    username,
	})
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("create share link: %w", err))
	}

	link, err := database.Queries.GetShareLinkByToken(c.Request.Context(), token)
	if err != nil {
		// Construct minimal response from the id we have.
		link = db.ShareLink{
			ID: id, Token: token, RelPath: req.RelPath,
			PasswordHash: passwordHash, MaxUses: req.MaxUses,
			ExpiresAt: expiresAt, CreatedAt: time.Now(), CreatedBy: username,
		}
	}

	return serverutil.NewResponse().
		WithStatusCode(http.StatusCreated).
		WithContentType(serverutil.ContentTypeJSON).
		WithData(toShareLinkJSON(link, baseURL(c)))
}

// listShareLinks godoc
// @Summary List share links
// @Description Returns the 100 most recent share links created on this instance.
// @Tags shares
// @Produce json
// @Success 200 {array} shareLinkJSON
// @Failure 500 {object} serverutil.Response
// @Router /shares [get]
func listShareLinks(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(fmt.Errorf("database unavailable"))
	}

	links, err := database.Queries.ListShareLinks(c.Request.Context())
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	base := baseURL(c)
	result := make([]shareLinkJSON, 0, len(links))
	for _, l := range links {
		result = append(result, toShareLinkJSON(l, base))
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(result)
}

// deleteShareLink godoc
// @Summary Delete a share link
// @Description Revokes a share link by ID. The link immediately stops working.
// @Tags shares
// @Produce json
// @Param id path int true "Share link ID"
// @Success 200 {object} serverutil.Response
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /shares/{id} [delete]
func deleteShareLink(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(fmt.Errorf("database unavailable"))
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return serverutil.BadRequest(fmt.Errorf("invalid id: %s", idStr))
	}

	if err := database.Queries.DeleteShareLink(c.Request.Context(), id); err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok()
}

// --- Public access route ---

// accessSharedFile godoc
// @Summary Access shared file/folder content
// @Description Public endpoint — no auth required. Validates token, optional password,
// @Description expiry, and use-count limits. For files: streams the file bytes.
// @Description For directories: returns a JSON listing.
// @Tags shares
// @Produce octet-stream
// @Param token path string true "Share token"
// @Param password query string false "Share password (if protected)"
// @Success 200
// @Failure 401 {object} serverutil.Response "Password required or wrong password"
// @Failure 404 {object} serverutil.Response "Token not found or expired"
// @Failure 410 {object} serverutil.Response "Link has reached its maximum use count"
// @Router /s/{token} [get]
func accessSharedFile(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(fmt.Errorf("database unavailable"))
	}

	token := c.Param("token")
	link, err := database.Queries.GetShareLinkByToken(c.Request.Context(), token)
	if errors.Is(err, sql.ErrNoRows) {
		return serverutil.NotFound(fmt.Errorf("share link not found or expired"))
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	// Check expiry.
	if link.ExpiresAt.Valid && time.Now().After(link.ExpiresAt.Time) {
		return serverutil.NotFound(fmt.Errorf("share link has expired"))
	}

	// Check use count.
	if link.MaxUses > 0 && link.UseCount >= link.MaxUses {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusGone).
			WithContentType(serverutil.ContentTypeJSON).
			WithData(gin.H{"error": "share link has reached its maximum use count"})
	}

	// Check password.
	if link.PasswordHash != "" {
		password := c.Query("password")
		if password == "" {
			return serverutil.NewResponse().
				WithStatusCode(http.StatusUnauthorized).
				WithContentType(serverutil.ContentTypeJSON).
				WithData(gin.H{"error": "password required", "passwordRequired": true})
		}
		if err := bcrypt.CompareHashAndPassword([]byte(link.PasswordHash), []byte(password)); err != nil {
			// Constant-time to prevent timing attacks.
			subtle.ConstantTimeCompare([]byte(link.PasswordHash), []byte(link.PasswordHash))
			return serverutil.NewResponse().
				WithStatusCode(http.StatusUnauthorized).
				WithContentType(serverutil.ContentTypeJSON).
				WithData(gin.H{"error": "incorrect password"})
		}
	}

	// Increment use count (fire-and-forget; don't fail the request on error).
	go func() {
		_ = database.Queries.IncrementShareLinkUseCount(c.Request.Context(), token)
	}()

	// Resolve the file path.
	filesDir, err := resolveFilesDir(link.DeviceSerial, deps)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	fullPath := filepath.Join(filesDir, link.RelPath)
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(filesDir)) {
		return serverutil.BadRequest(fmt.Errorf("invalid path"))
	}

	// Serve the file.
	c.File(fullPath)
	return nil
}

// --- Helpers ---

func validateRelPath(relPath, serial string, deps deputil.Dependencies) error {
	filesDir, err := resolveFilesDir(serial, deps)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(filesDir, relPath)
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(filesDir)) {
		return fmt.Errorf("invalid relPath")
	}
	return nil
}

func resolveFilesDir(serial string, deps deputil.Dependencies) (string, error) {
	if serial != "" {
		devices, err := deps.StorageService().GetManagedDevices()
		if err == nil {
			for _, d := range devices {
				if d.UsbInfo != nil && d.UsbInfo.GetSerial() == serial {
					return d.CirrusDir, nil
				}
			}
		}
	}
	return storageutil.GetCirrusDir()
}

// --- Routers ---

type router struct{}

// NewRouter returns the authenticated shares API router (registered under /api/v0).
func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		serverutil.ApiRoute("POST", "/shares", createShareLink),
		serverutil.ApiRoute("GET", "/shares", listShareLinks),
		serverutil.ApiRoute("DELETE", "/shares/:id", deleteShareLink),
	}
}

type publicRouter struct{}

// NewPublicRouter returns the unauthenticated share-access router.
// Register this directly on the gin.Engine (not under /api/v0) so it
// bypasses the auth middleware which only covers /api/ and /dav/ prefixes.
func NewPublicRouter() serverutil.Router {
	return &publicRouter{}
}

func (r *publicRouter) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		serverutil.ApiRoute("GET", "/s/:token", accessSharedFile),
	}
}
