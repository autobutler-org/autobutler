package v0_vault

import (
	"database/sql"
	"errors"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

type vaultStatusResponse struct {
	Initialized     bool   `json:"initialized"`
	Locked          bool   `json:"locked"`
	AutoLockSeconds int64  `json:"autoLockSeconds"`
	StorageDevice   string `json:"storageDevice"`
	DeviceConnected bool   `json:"deviceConnected"`
	LockReason      string `json:"lockReason,omitempty"`
}

var vaultStatusRoute = serverutil.ApiRoute(
	"GET", "/vault/status", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		ctx := c.Request.Context()
		config, err := deps.VaultDB().Queries.GetVaultConfig(ctx)

		initialized := true
		autoLock := int64(900)
		if errors.Is(err, sql.ErrNoRows) {
			initialized = false
		} else if err != nil {
			return serverutil.InternalServerError(err)
		} else {
			autoLock = config.AutoLockSeconds
		}

		locked := deps.VaultSession().IsLocked()
		lockReason := deps.VaultSession().LockReason()

		storageDevice := "internal"
		deviceConnected := true
		if serial, locErr := deps.Database().Queries.GetVaultLocation(ctx); locErr == nil && serial != "" {
			storageDevice = serial
			device, devErr := deps.StorageService().FindManagedDeviceBySerial(serial)
			deviceConnected = devErr == nil && device != nil
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(vaultStatusResponse{
			Initialized:     initialized,
			Locked:          locked,
			AutoLockSeconds: autoLock,
			StorageDevice:   storageDevice,
			DeviceConnected: deviceConnected,
			LockReason:      lockReason,
		})
	},
)
