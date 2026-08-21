package v0_smb

import (
	"errors"

	"github.com/autobutler-org/quark/internal/smb"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// SmbStatusJSON is the response for GET /smb/status.
type SmbStatusJSON struct {
	Linux      bool   `json:"linux"`
	Installed  bool   `json:"installed"`
	Configured bool   `json:"configured"`
	Running    bool   `json:"running"`
	FilesDir   string `json:"filesDir"`
}

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
