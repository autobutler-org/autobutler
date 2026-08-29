package v0_smb

import (
	"errors"

	"github.com/autobutler-org/quark/internal/smb"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// setupSmb godoc
// @Summary Set up SMB/Samba network share
// @Description Installs and configures Samba so Quark files are accessible as a network drive. Requires the backend to be running as root.
// @Tags smb
// @Accept json
// @Produce json
// @Param body body object true "{user: string, password: string}"
// @Success 200 {object} SmbStatusJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 501 {object} serverutil.Response "Not Implemented (non-Linux)"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /smb/setup [post]
var setupSmbRoute = serverutil.ApiRoute(
	"POST", "/smb/setup", func(c *gin.Context) *serverutil.Response {
		if !smb.IsLinux() {
			return serverutil.NewResponse().
				WithStatusCode(501).
				WithError(errors.New("SMB setup is only supported on Linux"))
		}

		var req struct {
			User     string `json:"user"     binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(err)
		}

		if err := smb.Setup(req.User, req.Password); err != nil {
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
