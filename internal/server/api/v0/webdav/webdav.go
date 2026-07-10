package v0_webdav

import (
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/net/webdav"

	"github.com/gin-gonic/gin"
)

const pathPrefix = "/dav/"

// NewHandler creates a gin.HandlerFunc that serves WebDAV requests
// against the given filesystem root directory.
func NewHandler(fsRoot string) gin.HandlerFunc {
	davHandler := &webdav.Handler{
		Prefix:     "/dav",
		FileSystem: webdav.Dir(fsRoot),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				slog.Error("webdav", "method", r.Method, "path", r.URL.Path, "err", err)
			} else {
				slog.Debug("webdav", "method", r.Method, "path", r.URL.Path)
			}
		},
	}

	return func(c *gin.Context) {
		davHandler.ServeHTTP(c.Writer, c.Request)
	}
}

// WebDAVMethods returns all HTTP methods used by the WebDAV protocol.
func WebDAVMethods() []string {
	return []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPut,
		http.MethodPost,
		http.MethodDelete,
		http.MethodOptions,
		"PROPFIND",
		"PROPPATCH",
		"MKCOL",
		"COPY",
		"MOVE",
		"LOCK",
		"UNLOCK",
	}
}

// IsWebDAVPath returns true if the request path is under the WebDAV prefix.
func IsWebDAVPath(path string) bool {
	return strings.HasPrefix(path, pathPrefix) || path == "/dav"
}
