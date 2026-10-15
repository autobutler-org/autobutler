package v0_previews

import (
	"context"
	"log/slog"

	"github.com/autobutler-org/quark/pkg/util/docreadutil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// previewDocument godoc
// @Summary Preview a word-processing document
// @Description Streams a .docx, .docm or .odt document as NDJSON, one block per line, for the read-only document viewer: a meta line, then headings, paragraphs, list items, tables, images (naming the part GET /previews/part serves), page breaks and omitted markers, then an end line with word, character and paragraph counts, warnings, and truncated set when a limit stopped the document early. The end line is always last: a stream without one was cut off by a server error. Nothing the document contains is executed or fetched. A file that is not what its name says is a 400. Needs read access on the document.
// @Tags previews
// @Produce application/x-ndjson
// @Param filePath query string true "Path to the document"
// @Param serial query string false "Device serial number"
// @Success 200 {string} string "NDJSON block stream"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Failure 503 {object} serverutil.Response "Server busy converting other documents"
// @Security BearerAuth
// @Router /previews/document [get]
func previewDocument(c *gin.Context) *serverutil.Response {
	deps, filePath, serial, refusal := readableDocument(c)
	if refusal != nil {
		return refusal
	}
	release, busy := acquireDecode(c, deps)
	if busy != nil {
		return busy
	}
	defer release()

	ctx, cancel := context.WithTimeout(c.Request.Context(), previewDeadline)
	defer cancel()
	_, err := fileutil.PreviewDocument(fileutil.PreviewDocumentParams{
		Ctx:      ctx,
		Registry: deps.VFSRegistry(),
		FilePath: filePath,
		Serial:   serial,
		Out:      streamWriter{c: c, contentType: docreadutil.NDJSONContentType},
	})
	if err == nil {
		return nil
	}
	if c.Writer.Written() {
		// The status is committed; the missing end line tells the client.
		slog.Warn("previews: document stream stopped partway", "path", filePath, "err", err)
		return nil
	}
	slog.Error("previews: document preview failed", "path", filePath, "err", err)
	return previewError(err)
}

var previewDocumentRoute = serverutil.ApiRoute(
	"GET", "/previews/document", previewDocument,
)
