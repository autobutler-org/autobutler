package v0_previews

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/iosemutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// readableDocument reads the request's filePath and serial and checks that
// the caller may read the document. A response is returned when they may
// not, or the request is incomplete.
func readableDocument(c *gin.Context) (deps deputil.Dependencies, filePath, serial string, refusal *serverutil.Response) {
	filePath = c.Query("filePath")
	serial = c.Query("serial")
	if filePath == "" {
		return deps, "", "", serverutil.BadRequest(errors.New("filePath query parameter is required"))
	}
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return deps, "", "", serverutil.InternalServerError(nil)
	}
	access, err := accessutil.LoadRequest(c, deps.Database(), deps.StorageService())
	if err != nil {
		return deps, "", "", serverutil.InternalServerError(err)
	}
	if !access.Check(serial, filePath, accessutil.Read).Readable {
		return deps, "", "", serverutil.NotFound(errNoAccess)
	}
	return deps, filePath, serial, nil
}

// acquireDecode takes an IO semaphore Decode slot for a conversion, so a
// burst of opens queues rather than stacking parsers. It answers 503 when no
// slot frees up in time; otherwise the caller calls release when done.
func acquireDecode(c *gin.Context, deps deputil.Dependencies) (release func(), busy *serverutil.Response) {
	sem := deps.IOSemaphore().For(iosemutil.Decode)
	if sem == nil {
		return func() {}, nil
	}
	if !sem.AcquireDefault(c.Request.Context()) {
		slog.Warn("previews: IO semaphore timed out", "path", c.Query("filePath"), "available", sem.Available(), "cap", sem.Cap())
		c.Header("Retry-After", "5")
		return nil, serverutil.NewResponse().WithStatusCode(http.StatusServiceUnavailable).
			WithError(errors.New("server busy, please retry"))
	}
	return sem.Release, nil
}

// previewError maps what fileutil reports onto status codes: a document that
// is missing, or not one Quark can read, is the caller's, not the server's.
func previewError(err error) *serverutil.Response {
	var notFound *fileutil.NotFoundError
	if errors.As(err, &notFound) {
		return serverutil.NotFound(err)
	}
	var unsupported *fileutil.UnsupportedError
	if errors.As(err, &unsupported) {
		return serverutil.BadRequest(err)
	}
	return serverutil.InternalServerError(err)
}
