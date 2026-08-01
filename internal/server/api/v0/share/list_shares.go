package v0_share

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

type shareLinkSummary struct {
	Token        string  `json:"token"`
	ResourceType string  `json:"resourceType"`
	ResourcePath string  `json:"resourcePath"`
	DeviceSerial string  `json:"deviceSerial"`
	ExpiresAt    *string `json:"expiresAt,omitempty"`
	ViewCount    int64   `json:"viewCount"`
	CreatedAt    string  `json:"createdAt"`
}

// listShares godoc
// @Summary List share links
// @Description List all active share links created by the authenticated user.
// @Tags share
// @Produce json
// @Success 200 {array} shareLinkSummary
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /share [get]
func listShares(c *gin.Context) *serverutil.Response {
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

	user, err := database.Queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("resolve user: %w", err))
	}

	links, err := database.Queries.ListShareLinksByUser(c.Request.Context(), sql.NullInt64{Int64: user.ID, Valid: true})
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("list share links: %w", err))
	}

	summaries := make([]shareLinkSummary, 0, len(links))
	for _, l := range links {
		sum := shareLinkSummary{
			// We never expose the raw token hash — the token was returned only
			// at creation time. The token_hash serves as the opaque revocation key.
			Token:        l.TokenHash, // clients use this to revoke; not reconstructable
			ResourceType: l.ResourceType,
			ResourcePath: l.ResourcePath,
			DeviceSerial: l.DeviceSerial,
			ViewCount:    l.ViewCount,
			CreatedAt:    l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if l.ExpiresAt.Valid {
			s := l.ExpiresAt.Time.Format("2006-01-02T15:04:05Z")
			sum.ExpiresAt = &s
		}
		summaries = append(summaries, sum)
	}

	return serverutil.Ok().WithData(summaries)
}

var listSharesRoute = serverutil.ApiRoute("GET", "/share", listShares)
