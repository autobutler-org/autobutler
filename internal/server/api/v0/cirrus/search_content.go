package v0_files

import (
	"fmt"
	"net/http"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// ContentResultJSON is a single full-text search hit returned by the content
// search endpoint.
type ContentResultJSON struct {
	// Path is the relative file path within the Cirrus data directory.
	Path string `json:"path"`
	// Snippet is a short context extract around the matching terms, with HTML
	// <mark> tags wrapping matches. May be empty when the match is in metadata.
	Snippet string `json:"snippet,omitempty"`
}

// searchContent godoc
// @Summary Search file contents (full-text)
// @Description Searches the full-text index for documents whose contents match the given query. Uses SQLite FTS5 syntax: phrase="hello world", prefix=term*, boolean=AND/OR/NOT. Returns 503 when the FTS index is not initialised.
// @Tags cirrus
// @Produce json
// @Param q     query string true  "FTS5 query string"
// @Param limit query int    false "Maximum number of results (default 50)"
// @Success 200 {object} object{results=[]ContentResultJSON}
// @Failure 400 {object} serverutil.Response "Missing or empty query"
// @Failure 503 {object} serverutil.Response "FTS index not available"
// @Router /cirrus/search/content [get]
func searchContent(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
	}

	idx := deps.FTSIndex()
	if idx == nil {
		c.Header("Retry-After", "30")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "full-text index not available"})
		return nil
	}

	query := c.Query("q")
	if query == "" {
		return serverutil.BadRequest(fmt.Errorf("q is required"))
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil || limit <= 0 {
			limit = 50
		}
	}

	hits, err := idx.Search(c.Request.Context(), query, limit)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("search failed: %w", err))
	}

	results := make([]ContentResultJSON, 0, len(hits))
	for _, h := range hits {
		results = append(results, ContentResultJSON{
			Path:    h.Path,
			Snippet: h.Snippet,
		})
	}
	return serverutil.Ok().WithData(gin.H{"results": results})
}

var searchContentRoute = serverutil.ApiRoute("GET", "/cirrus/search/content", searchContent)
