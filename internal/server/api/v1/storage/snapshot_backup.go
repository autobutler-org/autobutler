package v1_storage

import (
	"context"
	"fmt"
	"log"

	"github.com/autobutler-org/autobutler/pkg/backup"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var snapshotStore = backup.NewInMemoryBackupJobStore()

// startSnapshotBackup godoc
// @Summary Start a snapshot backup to a device
// @Description Aggregates all files from all managed devices onto the target snapshot-backup device
// @Tags storage
// @Accept json
// @Produce json
// @Param body body object true "{targetDeviceSerial: string}"
// @Success 202 {object} object
// @Failure 400 {object} serverutil.Response
// @Failure 409 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /storage/devices/snapshot-backup [post]
func startSnapshotBackup(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		TargetDeviceSerial string `json:"targetDeviceSerial" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(fmt.Errorf("invalid request body: %w", err))
	}

	ctx := c.Request.Context()

	role, err := deps.Database().Queries.GetDeviceRole(ctx, req.TargetDeviceSerial)
	if err != nil || role != "snapshot-backup" {
		return serverutil.BadRequest(fmt.Errorf("target device must have the snapshot-backup role"))
	}

	// Check no backup is already running for this target.
	jobs, _ := snapshotStore.List(ctx)
	for _, j := range jobs {
		if j.TargetDeviceSerial == req.TargetDeviceSerial &&
			(j.Status == backup.BackupStatusPending ||
				j.Status == backup.BackupStatusScanning ||
				j.Status == backup.BackupStatusCopying) {
			return serverutil.NewResponse().WithStatusCode(409).WithError(
				fmt.Errorf("backup already running for this device (job %s)", j.ID),
			)
		}
	}

	// Find the target managed device.
	targetDev, err := deps.StorageService().FindManagedDeviceBySerial(req.TargetDeviceSerial)
	if err != nil || targetDev == nil {
		return serverutil.BadRequest(fmt.Errorf("target device not found or not managed"))
	}

	// Gather all source devices (everything that isn't the target).
	sources, err := gatherSourceDevices(ctx, deps, req.TargetDeviceSerial)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to gather sources: %w", err))
	}

	job := &backup.BackupJob{
		ID:                 uuid.New().String(),
		Status:             backup.BackupStatusPending,
		TargetDeviceSerial: req.TargetDeviceSerial,
	}
	if err := snapshotStore.Create(ctx, job); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to create job: %w", err))
	}

	go func() {
		if err := backup.SnapshotBackup(context.Background(), backup.SnapshotBackupParams{
			TargetDeviceSerial: req.TargetDeviceSerial,
			Job:                job,
			Store:              snapshotStore,
			EventBus:           deps.EventBus(),
		}, sources, targetDev); err != nil {
			log.Printf("snapshot backup failed: %v", err)
		}
	}()

	return serverutil.Accepted().WithData(gin.H{
		"jobId": job.ID,
	})
}

// getSnapshotBackupStatus godoc
// @Summary Get snapshot backup job status
// @Description Returns the current status of a snapshot backup job
// @Tags storage
// @Produce json
// @Param jobId path string true "Job ID"
// @Success 200 {object} object
// @Failure 404 {object} serverutil.Response
// @Router /storage/devices/snapshot-backup/status/{jobId} [get]
func getSnapshotBackupStatus(c *gin.Context) *serverutil.Response {
	jobID := c.Param("jobId")
	job, err := snapshotStore.Get(c.Request.Context(), jobID)
	if err != nil {
		return serverutil.NotFound(fmt.Errorf("job not found: %w", err))
	}
	return serverutil.Ok().WithData(job)
}

func gatherSourceDevices(ctx context.Context, deps deputil.Dependencies, targetSerial string) ([]backup.SourceDevice, error) {
	managed, err := deps.StorageService().GetManagedDevices()
	if err != nil {
		return nil, err
	}

	var sources []backup.SourceDevice
	for _, d := range managed {
		serial := ""
		name := d.Name
		if d.UsbInfo != nil {
			serial = d.UsbInfo.GetSerial()
		}
		if serial == targetSerial {
			continue
		}
		sources = append(sources, backup.SourceDevice{
			Name:      name,
			Serial:    serial,
			CirrusDir: d.CirrusDir,
		})
	}
	return sources, nil
}

var startSnapshotBackupRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/snapshot-backup", startSnapshotBackup,
)

var getSnapshotBackupStatusRoute = serverutil.ApiRoute(
	"GET", "/storage/devices/snapshot-backup/status/:jobId", getSnapshotBackupStatus,
)
