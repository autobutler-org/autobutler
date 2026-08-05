package v1_vfs

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// vfsMkdir godoc
// @Summary Create a directory in a VFS namespace
// @Description Creates the directory at the given path, including any missing parents (mkdir -p semantics).
// @Tags vfs
// @Produce json
// @Param ns path string true "Namespace ID"
// @Param path path string true "Directory path to create"
// @Success 201 "Created"
// @Failure 404 {object} serverutil.Response "Namespace not found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /vfs/{ns}/{path} [post]
func vfsMkdir(c *gin.Context) *serverutil.Response {
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

	if strings.HasSuffix(strings.TrimRight(path, "/"), "/_meta") {
		return serverutil.NotFound(fmt.Errorf("use the meta endpoint for /_meta paths"))
	}

	fsys, ok := reg.Get(ns)
	if !ok {
		return serverutil.NotFound(fmt.Errorf("namespace %q not found", ns))
	}

	if err := fsys.MkdirAll(c.Request.Context(), path); err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.NewResponse().WithStatusCode(http.StatusCreated)
}

var vfsMkdirRoute = serverutil.ApiRoute("POST", "/vfs/:ns/*path", vfsMkdir)
