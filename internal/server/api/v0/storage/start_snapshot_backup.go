package v0_storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/autobutler-org/quark/pkg/backup"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
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
		Username           string `json:"username"`
		Password           string `json:"password"`
		RecoveryPassword   string `json:"recoveryPassword"`
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

	// If a recovery password is provided, validate credentials and prepare vault export.
	var vaultParams *backup.VaultExportParams
	if req.RecoveryPassword != "" {
		if req.Username == "" || req.Password == "" {
			return serverutil.BadRequest(fmt.Errorf("username and password required for vault backup"))
		}
		if len(req.RecoveryPassword) < 8 {
			return serverutil.BadRequest(fmt.Errorf("recovery password must be at least 8 characters"))
		}

		if _, err := authutil.ValidateBasicAuth(ctx, deps.Database().Queries, req.Username, req.Password); err != nil {
			return serverutil.Unauthorized(fmt.Errorf("invalid credentials"))
		}

		config, err := deps.Database().Queries.GetVaultConfig(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return serverutil.BadRequest(fmt.Errorf("vault is not initialized"))
		}
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get vault config: %w", err))
		}

		params := vaultcrypto.Argon2Params{
			Memory:      uint32(config.Argon2Memory),
			Iterations:  uint32(config.Argon2Iterations),
			Parallelism: uint8(config.Argon2Parallelism),
		}
		liveKey := vaultcrypto.DeriveKey(req.Password, config.Salt, params)

		if !vaultcrypto.CheckVerificationBlob(liveKey, config.VerificationBlob, config.VerificationNonce) {
			vaultcrypto.ZeroKey(liveKey)
			return serverutil.Unauthorized(fmt.Errorf("master password does not match vault"))
		}

		vaultParams = &backup.VaultExportParams{
			Queries:          deps.Database().Queries,
			LiveKey:          liveKey,
			RecoveryPassword: req.RecoveryPassword,
		}
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
		if vaultParams != nil {
			defer vaultcrypto.ZeroKey(vaultParams.LiveKey)
		}
		if err := backup.SnapshotBackup(context.Background(), backup.SnapshotBackupParams{
			TargetDeviceSerial: req.TargetDeviceSerial,
			Job:                job,
			Store:              snapshotStore,
			EventBus:           deps.EventBus(),
			Vault:              vaultParams,
			IOSemaphore:        deps.IOSemaphore(),
		}, sources, targetDev); err != nil {
			log.Printf("snapshot backup failed: %v", err)
		}
	}()

	return serverutil.Accepted().WithData(gin.H{
		"jobId": job.ID,
	})
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
			Name:     name,
			Serial:   serial,
			FilesDir: d.FilesDir,
		})
	}
	return sources, nil
}

var startSnapshotBackupRoute = serverutil.ApiRoute(
	"POST", "/storage/devices/snapshot-backup", startSnapshotBackup,
)
