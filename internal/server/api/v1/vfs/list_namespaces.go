package v1_vfs

import (
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// listNamespaces godoc
// @Summary List VFS namespaces
// @Description Returns all registered VFS namespaces visible to the caller
// @Tags vfs
// @Produce json
// @Success 200 {array} NamespaceJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /vfs [get]
func listNamespaces(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	reg := deps.VFSRegistry()
	if reg == nil {
		return serverutil.Ok().WithData([]NamespaceJSON{})
	}
	namespaces := reg.List("")
	out := make([]NamespaceJSON, len(namespaces))
	for i, ns := range namespaces {
		out[i] = NamespaceJSON{ID: ns.ID, Description: ns.Description}
	}
	return serverutil.Ok().WithData(out)
}

var listNamespacesRoute = serverutil.ApiRoute("GET", "/vfs", listNamespaces)
