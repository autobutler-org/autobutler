package v1_vfs

import (
	"fmt"
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// vfsWrite godoc
// @Summary Write a file to a VFS namespace
// @Description Writes the request body as a file at the given path. Set Content-Type to declare the MIME type. Use If-None-Match: * to fail if the file already exists.
// @Tags vfs
// @Accept application/octet-stream
// @Produce json
// @Param ns path string true "Namespace ID"
// @Param path path string true "Destination path"
// @Success 201 {object} FileInfoJSON
// @Failure 409 {object} serverutil.Response "Conflict (file exists and If-None-Match: * was set)"
// @Failure 404 {object} serverutil.Response "Namespace not found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /vfs/{ns}/{path} [put]
func vfsWrite(c *gin.Context) *serverutil.Response {
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
		return handleSetMeta(c, deps, ns, realPath)
	}

	fsys, ok := reg.Get(ns)
	if !ok {
		return serverutil.NotFound(fmt.Errorf("namespace %q not found", ns))
	}

	opts := vfs.WriteOptions{
		ContentType: c.ContentType(),
		IfNoneMatch: c.GetHeader("If-None-Match"),
	}
	if err := fsys.Write(c.Request.Context(), path, c.Request.Body, opts); err != nil {
		if err == vfs.ErrConflict {
			return serverutil.NewResponse().WithStatusCode(http.StatusConflict).WithError(err)
		}
		return serverutil.InternalServerError(err)
	}

	fi, err := fsys.Stat(c.Request.Context(), path)
	if err != nil {
		// Write succeeded but stat failed — return 201 with minimal info.
		c.Status(http.StatusCreated)
		return nil
	}
	c.Status(http.StatusCreated)
	c.JSON(http.StatusCreated, fileInfoToJSON(fi))
	return nil
}

var vfsWriteRoute = serverutil.ApiRoute("PUT", "/vfs/:ns/*path", vfsWrite)
