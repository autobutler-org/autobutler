package v0_music

import (
	"fmt"
	"strconv"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// loadCaller reads the dependencies and the caller's access. A non-nil
// response is the failure to send.
func loadCaller(c *gin.Context) (caller, *serverutil.Response) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return caller{}, serverutil.InternalServerError(nil)
	}
	access, err := accessutil.LoadRequest(c, deps.Database(), deps.StorageService())
	if err != nil {
		return caller{}, serverutil.InternalServerError(err)
	}
	return caller{deps: deps, access: access}, nil
}

// queryInt reads a non-negative integer query parameter, 0 when absent.
func queryInt(c *gin.Context, name string) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return n, nil
}

// paging reads the offset and limit query parameters.
func paging(c *gin.Context) (offset, limit int, err error) {
	if offset, err = queryInt(c, "offset"); err != nil {
		return 0, 0, err
	}
	limit, err = queryInt(c, "limit")
	return offset, limit, err
}

// optionalQuery is a query parameter that filters when present, even when
// empty: artist= names the untagged artist.
func optionalQuery(c *gin.Context, name string) *string {
	if v, ok := c.GetQuery(name); ok {
		return &v
	}
	return nil
}

// jsonOk answers 200 with data as JSON.
func jsonOk(data any) *serverutil.Response {
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(data)
}
