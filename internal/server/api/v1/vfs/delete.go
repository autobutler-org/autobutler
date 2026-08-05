package v1_vfs

import (
	"fmt"
	"net/http"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// vfsDelete godoc
// @Summary Delete a file or directory from a VFS namespace
// @Description Deletes the entry at the given path. For non-empty directories, ?recursive=true is required.
// @Tags vfs
// @Produce json
// @Param ns path string true "Namespace ID"
// @Param path path string true "Path to delete"
// @Param recursive query bool false "Delete directory contents recursively"
// @Success 204 "Deleted"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 409 {object} serverutil.Response "Directory not empty (use ?recursive=true)"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /vfs/{ns}/{path} [delete]
func vfsDelete(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	reg := deps.VFSRegistry()
	if reg == nil {
		return serverutil.NotFound(fmt.Errorf("vfs not available"))
	}

	ns := c.Param("ns")
	path := c.Param("path")

	if realPath, ok := isMetaPath(path); ok {
		return handleDeleteMeta(c, deps, ns, realPath)
	}

	fsys, ok := reg.Get(ns)
	if !ok {
		return serverutil.NotFound(fmt.Errorf("namespace %q not found", ns))
	}

	opts := vfs.DeleteOptions{Recursive: c.Query("recursive") == "true"}
	if err := fsys.Delete(c.Request.Context(), path, opts); err != nil {
		switch err {
		case vfs.ErrNotFound:
			// Treat concurrent deletes as success.
			return serverutil.NewResponse().WithStatusCode(http.StatusNoContent)
		case vfs.ErrNotEmpty:
			return serverutil.NewResponse().WithStatusCode(http.StatusConflict).WithError(err)
		default:
			return serverutil.InternalServerError(err)
		}
	}
	return serverutil.NewResponse().WithStatusCode(http.StatusNoContent)
}

var vfsDeleteRoute = serverutil.ApiRoute("DELETE", "/vfs/:ns/*path", vfsDelete)
