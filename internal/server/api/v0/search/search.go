// Package v0_search implements the full-text document search API.
package v0_search

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/ftsutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/gin-gonic/gin"
)

// searchResponse wraps search results.
type searchResponse struct {
	Query   string                 `json:"query"`
	Results []ftsutil.SearchResult `json:"results"`
	Total   int                    `json:"total"`
}

// searchDocuments godoc
// @Summary Full-text document search
// @Description Searches indexed document contents using SQLite FTS5.
// @Description Supports FTS5 query syntax: AND, OR, NOT, phrase "exact match", prefix*.
// @Description Pass index=true to trigger background indexing of new/changed files first.
// @Tags search
// @Produce json
// @Param q query string true "FTS5 search query"
// @Param limit query int false "Max results (default 20, max 100)"
// @Param serial query string false "Device serial to scope search"
// @Param index query bool false "Trigger background re-index before search"
// @Success 200 {object} searchResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /search [get]
func searchDocuments(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	database := deps.Database()
	if database == nil {
		return serverutil.InternalServerError(fmt.Errorf("database unavailable"))
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		return serverutil.BadRequest(fmt.Errorf("q parameter is required"))
	}

	limit := 20
	if ls := c.Query("limit"); ls != "" {
		if n, err := strconv.Atoi(ls); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}

	// Optional: trigger re-index before search.
	if c.Query("index") == "true" {
		serial := c.Query("serial")
		go func() {
			if err := indexAll(c.Request.Context(), deps, serial); err != nil {
				// Non-fatal — log and proceed.
				_ = err
			}
		}()
	}

	results, err := ftsutil.Search(c.Request.Context(), database.Db, q, limit)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("search: %w", err))
	}
	if results == nil {
		results = []ftsutil.SearchResult{}
	}

	return serverutil.NewResponse().
		WithStatusCode(http.StatusOK).
		WithContentType(serverutil.ContentTypeJSON).
		WithData(searchResponse{
			Query:   q,
			Results: results,
			Total:   len(results),
		})
}

// indexDocuments godoc
// @Summary Trigger FTS document indexing
// @Description Walks the cirrus directory and indexes all supported files.
// @Description Returns immediately; indexing happens in the background.
// @Tags search
// @Produce json
// @Param serial query string false "Device serial (defaults to primary storage)"
// @Success 202 {object} object
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /search/index [post]
func indexDocuments(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	serial := c.Query("serial")
	go func() {
		_ = indexAll(c.Request.Context(), deps, serial)
	}()

	return serverutil.NewResponse().
		WithStatusCode(http.StatusAccepted).
		WithContentType(serverutil.ContentTypeJSON).
		WithData(gin.H{"message": "indexing started"})
}

// indexAll walks the cirrus directory and indexes all supported files.
func indexAll(ctx context.Context, deps deputil.Dependencies, serial string) error {
	database := deps.Database()
	if database == nil {
		return fmt.Errorf("database unavailable")
	}

	filesDir, err := resolveFilesDir(serial, deps)
	if err != nil {
		return err
	}

	return filepath.WalkDir(filesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !ftsutil.IsIndexable(d.Name()) {
			return nil
		}
		relPath, relErr := filepath.Rel(filesDir, path)
		if relErr != nil {
			return nil
		}
		// Non-fatal per file.
		_, _ = ftsutil.IndexFile(ctx, database.Queries, database.Db, serial, relPath, path)
		return nil
	})
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

// --- Router ---

type router struct{}

func NewRouter() serverutil.Router { return &router{} }

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		serverutil.ApiRoute("GET", "/search", searchDocuments),
		serverutil.ApiRoute("POST", "/search/index", indexDocuments),
	}
}
