package v1_storage

import (
	"autobutler/pkg/storage"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// setDeviceBackup godoc
// @Summary Mark a device as a backup device
// @Description Persistently mark a USB device (by serial) as a backup device
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/backup/{serial} [post]
func setDeviceBackup(c *gin.Context) *serverutil.Response {
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("`serial` path parameter is required"))
	}
	// persist flag
	if err := storageutil.SetDeviceBackup(serial, true); err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to persist backup flag: %w", err))
	}
	// create and start backup job
	jobID, err := storage.CreateBackupJob(serial)
	if err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to start backup job: %w", err))
	}
	return serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(gin.H{"message": "Device marked as backup", "jobId": jobID})
}

// unsetDeviceBackup godoc
// @Summary Unmark a device as a backup device
// @Description Remove backup marking for a USB device
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/backup/{serial} [delete]
func unsetDeviceBackup(c *gin.Context) *serverutil.Response {
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("`serial` path parameter is required"))
	}
	// cancel any jobs associated with this target serial
	if err := storage.CancelJobsForTarget(serial); err != nil {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusInternalServerError).
			WithError(fmt.Errorf("Failed to cancel backup jobs: %w", err))
	}
	return serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(gin.H{"message": "Backup jobs canceled for device"})
}

// getDeviceBackupStatus godoc
// @Summary Get backup job(s) status for a device
// @Description Returns status of backup job(s) associated with the given storage device serial
// @Tags storage
// @Produce json
// @Param serial path string true "Device serial"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /storage/devices/backup/{serial} [get]
func getDeviceBackupStatus(c *gin.Context) *serverutil.Response {
	serial := c.Param("serial")
	if serial == "" {
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusBadRequest).
			WithError(errors.New("`serial` path parameter is required"))
	}
	includeFiles := false
	if val := c.Query("includeFiles"); val == "true" || val == "1" {
		includeFiles = true
	}
	jobs, err := storage.GetJobsForTarget(serial)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	if !includeFiles {
		// return copies without the file lists
		out := make([]*storage.BackupJob, 0, len(jobs))
		for _, j := range jobs {
			cp := *j
			cp.Files = nil
			out = append(out, &cp)
		}
		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(out)
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(jobs)
}

var setDeviceBackupRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/backup/:serial", setDeviceBackup,
)

var unsetDeviceBackupRoute = serverutil.ApiRoute(
	"DELETE", "/storage/devices/backup/:serial", unsetDeviceBackup,
)

var getDeviceBackupStatusRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/backup/:serial", getDeviceBackupStatus,
)
