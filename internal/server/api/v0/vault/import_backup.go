package v0_vault

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/backup"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

type importBackupRequest struct {
	DeviceSerial     string `json:"deviceSerial" binding:"required"`
	RecoveryPassword string `json:"recoveryPassword" binding:"required"`
}

var importBackupRoute = serverutil.ApiRoute(
	"POST", "/vault/import-backup", func(c *gin.Context) *serverutil.Response {
		deps, liveKey, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(liveKey)

		var req importBackupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		dev, err := deps.StorageService().FindManagedDeviceBySerial(req.DeviceSerial)
		if err != nil || dev == nil {
			return serverutil.BadRequest(fmt.Errorf("device not found or not managed"))
		}

		ctx := c.Request.Context()

		tx, err := deps.VaultDB().Db.BeginTx(ctx, nil)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("begin tx: %w", err))
		}
		defer tx.Rollback()

		qtx := deps.VaultDB().Queries.WithTx(tx)

		result, err := backup.ImportVault(ctx, qtx, liveKey, req.RecoveryPassword, dev.CirrusDir)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("import failed: %w", err))
		}

		if err := tx.Commit(); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("commit: %w", err))
		}

		return serverutil.Ok().WithData(result)
	},
)
