package frame

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed assets
var assetFS embed.FS

// contentTypes names the assets by extension; a file with another extension is not served.
var contentTypes = map[string]string{
	".html":        "text/html; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".webmanifest": "application/manifest+json",
}

// serveAsset answers /frame and /frame/<name> from the embedded assets. The bare path and a trailing slash are the
// page itself. Assets revalidate on every load through an ETag, so a reload on a slow link costs one small 304.
func serveAsset(c *gin.Context) {
	name := strings.TrimPrefix(path.Clean("/"+c.Param("asset")), "/")
	if name == "" {
		name = "index.html"
	}
	contentType, ok := contentTypes[path.Ext(name)]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	body, err := fs.ReadFile(assetFS, path.Join("assets", name))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Type", contentType)
	c.Header("ETag", fmt.Sprintf(`"%x"`, sha256.Sum256(body)))
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "no-referrer")
	// A zero modification time leaves validation to the ETag.
	http.ServeContent(c.Writer, c.Request, name, time.Time{}, bytes.NewReader(body))
}
