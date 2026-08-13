package v0_files

import (
	"strconv"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/searchutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// ContentSearchResult is the JSON shape for a single content search hit.
type ContentSearchResult struct {
	// Serial is the storage device serial number.
	Serial string `json:"serial"`
	// RelPath is the path to the file relative to the device's CirrusDir.
	RelPath string `json:"relPath"`
	// Snippet is a highlighted excerpt around the matched terms.
	// Matched terms are enclosed in <b> tags.
	Snippet string `json:"snippet"`
}

// searchContent godoc
// @Summary Search file contents (FTS5)
// @Schemes http https
// @Description Full-text search over indexed file contents using SQLite FTS5.
// @Description Only text-based file formats are indexed (.txt, .md, .csv, .yaml, .json, etc.).
// @Description Binary formats (images, video, PDF) are not indexed.
// @Description Returns up to `limit` results ordered by relevance rank.
// @Tags cirrus
// @Produce json
// @Param q      query string true  "Search query (FTS5 syntax: words, AND/OR/NOT, \"phrases\", prefix*)"
// @Param limit  query int    false "Maximum results to return (default 50, max 200)"
// @Success 200 {array}  ContentSearchResult
// @Failure 400 {object} serverutil.Response "Bad Request — missing query"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/search/content [get]
func searchContent(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		return serverutil.BadRequest(nil)
	}

	limit := searchutil.DefaultLimit
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}

	dbConn := deps.Database()
	if dbConn == nil || dbConn.Db == nil {
		return serverutil.InternalServerError(nil)
	}

	results, err := searchutil.Search(c.Request.Context(), dbConn.Db, q, limit)
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	out := make([]ContentSearchResult, len(results))
	for i, r := range results {
		out[i] = ContentSearchResult{
			Serial:  r.Serial,
			RelPath: r.RelPath,
			Snippet: r.Snippet,
		}
	}
	return serverutil.Ok().WithData(out)
}

var searchContentRoute = serverutil.ApiRoute(
	"GET", "/cirrus/search/content",
	func(c *gin.Context) *serverutil.Response { return searchContent(c) },
)
