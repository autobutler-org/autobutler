package v0_files

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// xlsxContentType is the media type of an .xlsx workbook.
const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// exportXlsx godoc
// @Summary Download a Quark spreadsheet as an Excel workbook
// @Description Streams a .qsheet back as an .xlsx attachment, one worksheet per tab in tab order. Values, bold, italic, colors, alignment, number formats, column widths and frozen panes carry across; formulas are written as their text, since the editor's formula dialect is not Excel's and the server has no evaluator to supply the cached results Excel expects. Only reads: nothing is written beside the sheet. Needs read access on the .qsheet.
// @Tags files
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param filePath query string true "Path to the .qsheet file to export"
// @Param serial query string false "Device serial number"
// @Success 200 {file} file "The workbook"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /files/export/xlsx [get]
func exportXlsx(c *gin.Context) *serverutil.Response {
	filePath := c.Query("filePath")
	if filePath == "" {
		return serverutil.BadRequest(errors.New("filePath query parameter is required"))
	}
	serial := c.Query("serial")

	deps, access, resp := loadCaller(c)
	if resp != nil {
		return resp
	}
	if !access.Check(serial, filePath, accessutil.Read).Readable {
		return serverutil.NotFound(errNoAccess)
	}
	return writeExport(c, filePath, xlsxContentType, fileutil.XlsxExportName(filePath), "export: xlsx export stopped partway", func() error {
		_, err := fileutil.ExportQsheetToXlsx(fileutil.ExportXlsxParams{
			Ctx:      c.Request.Context(),
			Registry: deps.VFSRegistry(),
			FilePath: filePath,
			Serial:   serial,
			Out:      c.Writer,
		})
		return err
	})
}

var exportXlsxRoute = serverutil.ApiRoute(
	"GET", "/files/export/xlsx", exportXlsx,
)
