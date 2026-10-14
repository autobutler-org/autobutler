package server

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

// compressibleExts are the web build's text and wasm files, the ones that
// shrink: main.dart.js by ~70%, canvaskit.wasm by ~60% (#3208). Images and
// woff2 fonts are compressed already.
var compressibleExts = map[string]bool{
	".css":  true,
	".html": true,
	".js":   true,
	".json": true,
	".mjs":  true,
	".svg":  true,
	".wasm": true,
}

// serveWebApp serves the web build in fsys: a file when the path names one,
// and index.html for every other path so Flutter's router can read the URL.
//
// Every response carries a content-hash ETag and Cache-Control: no-cache. No
// file in a Flutter build has a hashed name (main.dart.js is main.dart.js in
// every release), so a long max-age would keep an old app running after an
// upgrade; instead the browser revalidates and gets a 304 with no body. The
// ETag is weak because the same file goes out as identity, gzip or brotli.
//
// Compressible files are compressed as they stream, by Accept-Encoding, so
// nothing is held in memory; a Range request gets the identity bytes.
func serveWebApp(fsys fs.FS) gin.HandlerFunc {
	var etags sync.Map // file name → ETag
	etagOf := func(name string) (string, error) {
		if tag, ok := etags.Load(name); ok {
			return tag.(string), nil
		}
		f, err := fsys.Open(name)
		if err != nil {
			return "", err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		tag := `W/"` + hex.EncodeToString(h.Sum(nil)[:16]) + `"`
		etags.Store(name, tag)
		return tag, nil
	}

	return func(c *gin.Context) {
		name := strings.TrimPrefix(path.Clean("/"+c.Request.URL.Path), "/")
		if info, err := fs.Stat(fsys, name); err != nil || info.IsDir() {
			name = "index.html"
		}
		etag, err := etagOf(name)
		if err != nil {
			// No built frontend (dev mode): there is no index.html to fall back to.
			c.String(http.StatusNotFound, "404 page not found")
			return
		}
		f, err := fsys.Open(name)
		if err != nil {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}
		defer f.Close()

		w, r := c.Writer, c.Request
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", etag)
		ext := path.Ext(name)
		encoding := ""
		if compressibleExts[ext] && r.Header.Get("Range") == "" {
			encoding = acceptedEncoding(r.Header.Get("Accept-Encoding"))
			w.Header().Set("Vary", "Accept-Encoding")
		}
		if encoding == "" {
			// ServeContent answers If-None-Match itself, and Range.
			http.ServeContent(w, r, name, time.Time{}, f.(io.ReadSeeker))
			return
		}
		if inm := r.Header.Get("If-None-Match"); inm == "*" || strings.Contains(inm, strings.TrimPrefix(etag, "W/")) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", mime.TypeByExtension(ext))
		w.Header().Set("Content-Encoding", encoding)
		w.WriteHeader(http.StatusOK) // NoRoute starts every response at 404
		var cw io.WriteCloser = gzip.NewWriter(w)
		if encoding == "br" {
			// ponytail: compressed per request; level 5 does canvaskit.wasm in
			// ~0.7 s on x86, and an ETag hit skips it. Precompress at build time
			// if cold loads on a slow board cost too much CPU.
			cw = brotli.NewWriterLevel(w, 5)
		}
		defer cw.Close()
		if r.Method == http.MethodHead {
			return
		}
		_, _ = io.Copy(cw, f)
	}
}

// acceptedEncoding picks br, else gzip, from an Accept-Encoding header,
// honoring q=0; "" means send the identity bytes. (brotli.HTTPCompressor does
// this too, but its gzip writer panics on repetitive input.)
func acceptedEncoding(header string) string {
	accepted := map[string]bool{}
	for _, part := range strings.Split(header, ",") {
		coding, params, _ := strings.Cut(part, ";")
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			q, _ = strconv.ParseFloat(v, 64)
		}
		accepted[strings.ToLower(strings.TrimSpace(coding))] = q > 0
	}
	for _, encoding := range []string{"br", "gzip"} {
		if accepted[encoding] {
			return encoding
		}
	}
	return ""
}
