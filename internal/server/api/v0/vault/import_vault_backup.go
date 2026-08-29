package v0_vault

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var importVaultBackupRoute = serverutil.ApiRoute(
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

		result, err := vaultutil.ImportBackup(c.Request.Context(), vaultutil.ImportBackupParams{
			VaultDB:          deps.VaultDB(),
			LiveKey:          liveKey,
			RecoveryPassword: req.RecoveryPassword,
			BackupDir:        dev.FilesDir,
		})
		if errors.Is(err, vaultutil.ErrBackupImportFailed) {
			return serverutil.BadRequest(err)
		}
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithData(result.Import)
	},
)
