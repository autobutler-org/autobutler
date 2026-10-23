package v0_files

import (
	"errors"
	"log/slog"
	"path"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// convertDocument godoc
// @Summary Convert a document to a Quark document
// @Description Reads a .docx, .docm or .odt document and writes it as a sibling .qdoc, the format the document editor opens ("Open as Quark doc"). The document itself is left untouched. What a .qdoc cannot hold is flattened or left out and named in warnings: a table becomes one line per row with tab-separated cells, and images are dropped. Answers 409 when a .qdoc of that name already exists and overwrite was not asked for; with overwrite on the internal device, the replaced document is first snapshotted into its version history. A file that is not what its name says is a 400. Needs read access on the document and write access on its directory; the caller owns a newly created .qdoc.
// @Tags files
// @Produce json
// @Param filePath query string true "Path to the .docx, .docm or .odt file to convert"
// @Param serial query string false "Device serial number"
// @Param overwrite query bool false "Replace an existing .qdoc of the same name"
// @Success 200 {object} ConvertDocumentJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 403 {object} serverutil.Response "Forbidden"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 409 {object} serverutil.Response "Conflict"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /files/convert/document [post]
func convertDocument(c *gin.Context) *serverutil.Response {
	filePath := c.Query("filePath")
	if filePath == "" {
		return serverutil.BadRequest(errors.New("filePath query parameter is required"))
	}
	serial := c.Query("serial")

	deps, access, resp := loadCaller(c)
	if resp != nil {
		return resp
	}
	if resp := gateReadWrite(access, serial, filePath, path.Dir(filePath)); resp != nil {
		return resp
	}

	result, err := fileutil.ConvertDocumentToQdoc(fileutil.ConvertDocumentParams{
		Ctx:             c.Request.Context(),
		Registry:        deps.VFSRegistry(),
		EventBus:        deps.EventBus(),
		FilePath:        filePath,
		Serial:          serial,
		Overwrite:       c.Query("overwrite") == "true",
		BeforeOverwrite: beforeSave(c, deps, serial),
	})
	if err != nil {
		slog.Error("convert: document conversion failed", "path", filePath, "err", err)
		if errors.Is(err, vfs.ErrConflict) {
			return serverutil.Conflict(err)
		}
		return fileError(err)
	}
	if !result.Replaced {
		accessutil.RecordOwner(c.Request.Context(), deps.Database(), access, serial, result.Path)
	}

	warnings := result.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(ConvertDocumentJSON{
		Path:      result.Path,
		Truncated: result.Truncated,
		Words:     result.Words,
		Warnings:  warnings,
	})
}

var convertDocumentRoute = serverutil.ApiRoute(
	"POST", "/files/convert/document", convertDocument,
)
