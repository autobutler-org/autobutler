package v0_smb

import (
	"github.com/autobutler-org/quark/internal/smb"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// getSmbStatus godoc
// @Summary Get SMB/Samba network share status
// @Description Returns the current state of the Samba network share configuration
// @Tags smb
// @Produce json
// @Success 200 {object} SmbStatusJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /smb/status [get]
var getSmbStatusRoute = serverutil.ApiRoute(
	"GET", "/smb/status", func(c *gin.Context) *serverutil.Response {
		status, err := smb.GetStatus()
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(SmbStatusJSON{
			Linux:      status.Linux,
			Installed:  status.Installed,
			Configured: status.Configured,
			Running:    status.Running,
			FilesDir:   status.FilesDir,
		})
	},
)
