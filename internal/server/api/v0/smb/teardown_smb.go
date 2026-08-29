package v0_smb

import (
	"errors"

	"github.com/autobutler-org/quark/internal/smb"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// teardownSmb godoc
// @Summary Disable SMB/Samba network share
// @Description Removes the Quark share from smb.conf and stops the Samba service. Requires the backend to be running as root.
// @Tags smb
// @Produce json
// @Success 200 {object} SmbStatusJSON
// @Failure 501 {object} serverutil.Response "Not Implemented (non-Linux)"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /smb [delete]
var teardownSmbRoute = serverutil.ApiRoute(
	"DELETE", "/smb", func(c *gin.Context) *serverutil.Response {
		if !smb.IsLinux() {
			return serverutil.NewResponse().
				WithStatusCode(501).
				WithError(errors.New("SMB teardown is only supported on Linux"))
		}

		if err := smb.Teardown(); err != nil {
			return serverutil.InternalServerError(err)
		}

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
