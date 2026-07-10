package v1_vfs

// meta.go — /_meta endpoint logic.
//
// The VFS REST router uses a single wildcard route per method (/vfs/:ns/*path).
// Each CRUD handler (read, write, delete) checks whether the path ends with
// the reserved suffix "/_meta" and, if so, delegates here instead of
// touching the underlying VFS.
//
// The query endpoint gets its own dedicated static route registered in v1_vfs.go
// because it does not share the /:ns/*path wildcard shape.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

const metaSuffix = "/_meta"

// isMetaPath reports whether path ends with the /_meta reserved segment.
// It also returns the real file path with the suffix stripped.
func isMetaPath(path string) (realPath string, ok bool) {
	trimmed := strings.TrimRight(path, "/")
	if strings.HasSuffix(trimmed, metaSuffix) {
		return strings.TrimSuffix(trimmed, metaSuffix), true
	}
	return "", false
}

// handleGetMeta handles GET /vfs/:ns/*path when path ends with /_meta.
func handleGetMeta(c *gin.Context, deps deputil.Dependencies, ns, realPath string) *serverutil.Response {
	store := deps.MetadataStore()
	if store == nil {
		return serverutil.InternalServerError(fmt.Errorf("metadata store not available"))
	}
	meta, err := store.Get(c.Request.Context(), ns, realPath)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithData(meta)
}

// handleSetMeta handles PUT /vfs/:ns/*path when path ends with /_meta.
func handleSetMeta(c *gin.Context, deps deputil.Dependencies, ns, realPath string) *serverutil.Response {
	store := deps.MetadataStore()
	if store == nil {
		return serverutil.InternalServerError(fmt.Errorf("metadata store not available"))
	}
	var kv map[string]json.RawMessage
	if err := c.ShouldBindJSON(&kv); err != nil {
		return serverutil.BadRequest(err)
	}
	if err := store.Set(c.Request.Context(), ns, realPath, kv); err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok()
}

// handleDeleteMeta handles DELETE /vfs/:ns/*path when path ends with /_meta.
func handleDeleteMeta(c *gin.Context, deps deputil.Dependencies, ns, realPath string) *serverutil.Response {
	store := deps.MetadataStore()
	if store == nil {
		return serverutil.InternalServerError(fmt.Errorf("metadata store not available"))
	}
	var keys []string
	if err := c.ShouldBindJSON(&keys); err != nil {
		return serverutil.BadRequest(err)
	}
	if err := store.DeleteKeys(c.Request.Context(), ns, realPath, keys); err != nil {
		return serverutil.InternalServerError(err)
	}
	return &serverutil.Response{StatusCode: 204}
}

// queryMeta godoc
// @Summary Query metadata entries by key/value across a namespace
// @Description Returns all (namespace, path) entries where the given key exists (and optionally matches a specific value).
// @Tags vfs
// @Produce json
// @Param ns path string true "Namespace ID"
// @Param key query string true "Metadata key to match"
// @Param value query string false "Exact JSON-encoded value to match (optional; omit to match any entry with the key)"
// @Success 200 {array} object "Array of MetaEntry objects (namespace, path, meta map)"
// @Failure 400 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /vfs/{ns}/_meta/query [get]
func queryMeta(c *gin.Context) *serverutil.Response {
	deps, ok := c.Get("deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	d, ok := deps.(deputil.Dependencies)
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	store := d.MetadataStore()
	if store == nil {
		return serverutil.InternalServerError(fmt.Errorf("metadata store not available"))
	}

	ns := c.Param("ns")
	key := c.Query("key")
	if key == "" {
		return serverutil.BadRequest(fmt.Errorf("key query parameter is required"))
	}

	var value json.RawMessage
	if v := c.Query("value"); v != "" {
		value = json.RawMessage(v)
	}

	entries, err := store.Query(c.Request.Context(), ns, key, value)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithData(entries)
}

var vfsQueryMetaRoute = serverutil.ApiRoute("GET", "/vfs/:ns/_meta/query", queryMeta)
