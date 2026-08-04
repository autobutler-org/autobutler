package v0_search

import (
	"errors"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// searchDocuments godoc
// @Summary Full-text search over document contents
// @Description Search indexed .abdoc, .absheet, and plain-text files using FTS5.
// @Description Supports plain words, "exact phrases", and boolean (word1 AND word2).
// @Description Returns up to 50 results ordered by relevance with HTML snippet excerpts.
// @Tags search
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {object} serverutil.Response "Search results"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /search/documents [get]
func searchDocuments(c *gin.Context) *serverutil.Response {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		return serverutil.BadRequest(errors.New("q query parameter is required"))
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	idx := deps.SearchIndex()
	if idx == nil {
		return serverutil.InternalServerError(errors.New("search index not initialized"))
	}

	results, err := idx.Search(c.Request.Context(), q)
	if err != nil {
		// FTS5 MATCH syntax errors surface as plain DB errors; return 400 so the
		// client can show "invalid query" rather than a generic 500.
		if strings.Contains(err.Error(), "fts5") || strings.Contains(err.Error(), "syntax") {
			return serverutil.BadRequest(err)
		}
		return serverutil.InternalServerError(err)
	}

	return serverutil.Ok().WithData(results)
}

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		serverutil.ApiRoute("GET", "/search/documents", searchDocuments),
	}
}
