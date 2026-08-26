package v0_files

import (
	"strings"

	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// TODO(pre-v1.0.0, #1601): delete this file, legacy_cirrus_alias_test.go, the
// legacyAliasRoutes call in v1_files.go, and the "/api/v0/cirrus" entry in
// queryTokenPrefixes in internal/server/middleware/middleware.go.
//
// Deprecated compatibility shim for the Cirrus -> Files rename (#1601).
//
// The API moved from /api/v0/cirrus/* to /api/v0/files/*. Clients built before
// that rename still call the old paths, so every canonical route is mirrored
// under /cirrus for a release.
//
// The alias serves the same handler rather than issuing a redirect: a 3xx would
// drop the request body on POST/PUT/DELETE in several HTTP clients, and it would
// cost downloads an extra round trip on every Range request.
//
// Responses served through an alias carry Deprecation and Link headers, so old
// clients are visible in access logs and proxies.
//
// To retire the shim: delete this file and its test, and drop the
// "/api/v0/cirrus" entry from queryTokenPrefixes in
// internal/server/middleware/middleware.go.
const (
	// canonicalPathPrefix is the path prefix every route in this package uses.
	canonicalPathPrefix = "/files"
	// legacyPathPrefix is the pre-rename prefix kept alive for old clients.
	legacyPathPrefix = "/cirrus"
)

// legacyAliasRoutes mirrors each canonical route under the legacy prefix,
// reusing the canonical handler. Route paths are preserved verbatim apart from
// the prefix swap, including the double slash in /files//upload/*rootDir.
func legacyAliasRoutes(canonical []*serverutil.Route) []*serverutil.Route {
	aliases := make([]*serverutil.Route, 0, len(canonical))
	for _, route := range canonical {
		if !strings.HasPrefix(route.Path, canonicalPathPrefix) {
			// Not a /files route, so there is no pre-rename path to alias.
			continue
		}
		aliases = append(aliases, &serverutil.Route{
			Method:  route.Method,
			Path:    legacyPathPrefix + strings.TrimPrefix(route.Path, canonicalPathPrefix),
			Handler: withDeprecationHeaders(route.Handler),
		})
	}
	return aliases
}

// withDeprecationHeaders announces the successor path before delegating to the
// canonical handler. The headers must be set first: handlers write their own
// status code and stream bodies, and headers set after that never reach the
// client.
func withDeprecationHeaders(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Deprecation", "true")
		c.Header("Link", "<"+successorPath(c.Request.URL.Path)+`>; rel="successor-version"`)
		handler(c)
	}
}

// successorPath rewrites the requested legacy path to its canonical equivalent.
// Only the first occurrence is replaced, so a file named "cirrus" further down
// the path is left alone.
func successorPath(requestPath string) string {
	return strings.Replace(requestPath, legacyPathPrefix, canonicalPathPrefix, 1)
}
