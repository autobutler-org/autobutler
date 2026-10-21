package v0_previews

import (
	"log/slog"

	"github.com/autobutler-org/quark/pkg/util/docpropsutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// getProperties godoc
// @Summary Get a document's properties
// @Description Returns what a document says about itself: title, author, subject, created and modified dates, and the page, word, character, slide and sheet counts its authoring application last saved, as {title, author, subject, created, modified, pages, words, characters, slides, sheets, application, countsSource}. Fields the document does not carry are left out, and countsSource is always "saved": the counts are not recounted, and pages cannot be, since that needs a layout engine. Only the small metadata entry is read (docProps for DOCX, PPTX and XLSX, meta.xml for ODT, ODP and ODS, the \info group for RTF), never the body. A file that is not what its name says, or whose metadata is oversize, is a 400. Needs read access on the document.
// @Tags previews
// @Produce json
// @Param filePath query string true "Path to the document"
// @Param serial query string false "Device serial number"
// @Success 200 {object} docpropsutil.Properties
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /previews/properties [get]
func getProperties(c *gin.Context) *serverutil.Response {
	deps, filePath, serial, refusal := readableDocument(c)
	if refusal != nil {
		return refusal
	}
	result, err := docpropsutil.Read(docpropsutil.ReadParams{
		Ctx:      c.Request.Context(),
		Registry: deps.VFSRegistry(),
		FilePath: filePath,
		Serial:   serial,
	})
	if err != nil {
		slog.Warn("previews: reading document properties failed", "path", filePath, "err", err)
		return previewError(err)
	}
	return serverutil.Ok().WithData(result.Properties)
}

var getPropertiesRoute = serverutil.ApiRoute(
	"GET", "/previews/properties", getProperties,
)
