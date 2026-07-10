package v1_vfs

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// vfsRead godoc
// @Summary Read or list a VFS path
// @Description If the path ends with '/' or the ?stat=true query is absent and the entry is a directory, returns a JSON array of FileInfoJSON. If the path is a file, streams the raw bytes. If ?stat=true is set, returns a single FileInfoJSON regardless.
// @Tags vfs
// @Produce json,application/octet-stream
// @Param ns path string true "Namespace ID"
// @Param path path string true "Path within the namespace (leading slash optional)"
// @Param stat query bool false "Return FileInfo JSON instead of streaming bytes"
// @Param recursive query bool false "List recursively (directory listing only)"
// @Param mime_prefix query string false "Filter by MIME type prefix (directory listing only)"
// @Success 200 {array} FileInfoJSON "Directory listing"
// @Success 200 {file} file "File bytes"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /vfs/{ns}/{path} [get]
func vfsRead(c *gin.Context) *serverutil.Response {
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
	if path == "" {
		path = "/"
	}

	// _meta suffix is handled by the meta handler (Phase 2b).
	if strings.HasSuffix(strings.TrimRight(path, "/"), "/_meta") {
		return serverutil.NotFound(fmt.Errorf("use the meta endpoint for /_meta paths"))
	}

	fsys, ok := reg.Get(ns)
	if !ok {
		return serverutil.NotFound(fmt.Errorf("namespace %q not found", ns))
	}

	statOnly := c.Query("stat") == "true"
	isDir := strings.HasSuffix(path, "/")

	if statOnly {
		fi, err := fsys.Stat(c.Request.Context(), path)
		if err != nil {
			return serverutil.NotFound(err)
		}
		return serverutil.Ok().WithData(fileInfoToJSON(fi))
	}

	if isDir || path == "/" {
		filter := &vfs.ListFilter{
			Recursive:  c.Query("recursive") == "true",
			MimePrefix: c.Query("mime_prefix"),
		}
		entries, err := fsys.List(c.Request.Context(), path, filter)
		if err != nil {
			if err == vfs.ErrNotFound {
				return serverutil.NotFound(err)
			}
			return serverutil.InternalServerError(err)
		}
		out := make([]FileInfoJSON, len(entries))
		for i, fi := range entries {
			out[i] = fileInfoToJSON(fi)
		}
		return serverutil.Ok().WithData(out)
	}

	// File: stat to get MIME type, then stream bytes.
	fi, err := fsys.Stat(c.Request.Context(), path)
	if err != nil {
		return serverutil.NotFound(err)
	}
	if fi.IsDir {
		// Path didn't have trailing slash but is actually a directory; list it.
		entries, err := fsys.List(c.Request.Context(), path+"/", &vfs.ListFilter{})
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		out := make([]FileInfoJSON, len(entries))
		for i, e := range entries {
			out[i] = fileInfoToJSON(e)
		}
		return serverutil.Ok().WithData(out)
	}

	rc, err := fsys.Open(c.Request.Context(), path)
	if err != nil {
		return serverutil.NotFound(err)
	}
	defer rc.Close()

	mimeType := fi.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	c.Header("Content-Type", mimeType)
	if fi.Size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", fi.Size))
	}
	c.Status(http.StatusOK)
	io.Copy(c.Writer, rc) //nolint:errcheck
	return nil            // response written directly
}

var vfsReadRoute = serverutil.ApiRoute("GET", "/vfs/:ns/*path", vfsRead)

func fileInfoToJSON(fi vfs.FileInfo) FileInfoJSON {
	return FileInfoJSON{
		Name:        fi.Name,
		Path:        fi.Path,
		Size:        fi.Size,
		IsDir:       fi.IsDir,
		MimeType:    fi.MimeType,
		ModTime:     fi.ModTime,
		ContentHash: fi.ContentHash,
		Namespace:   fi.Namespace,
	}
}
