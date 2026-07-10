package v0_vault

import (
	"fmt"
	"path/filepath"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/backup"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

type storageLocationResponse struct {
	DeviceSerial    string `json:"deviceSerial"`
	IsExternal      bool   `json:"isExternal"`
	DeviceConnected bool   `json:"deviceConnected"`
	DeviceName      string `json:"deviceName"`
}

// getStorageLocation godoc
// @Summary Get vault storage location
// @Description Returns which device the vault is stored on
// @Tags vault
// @Produce json
// @Success 200 {object} storageLocationResponse
// @Failure 500 {object} serverutil.Response
// @Router /vault/storage-location [get]
var getStorageLocationRoute = serverutil.ApiRoute(
	"GET", "/vault/storage-location", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		ctx := c.Request.Context()
		serial, err := deps.Database().Queries.GetVaultLocation(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get vault location: %w", err))
		}

		resp := storageLocationResponse{
			DeviceSerial: serial,
			IsExternal:   serial != "",
		}

		if serial == "" {
			resp.DeviceConnected = true
			resp.DeviceName = "Internal Storage"
		} else {
			device, err := deps.StorageService().FindManagedDeviceBySerial(serial)
			if err == nil && device != nil {
				resp.DeviceConnected = true
				resp.DeviceName = device.Name
				if name, err := deps.Database().Queries.GetDeviceName(ctx, serial); err == nil && name != "" {
					resp.DeviceName = name
				}
			}
		}

		return serverutil.Ok().WithData(resp)
	},
)

type setStorageLocationRequest struct {
	TargetDeviceSerial string `json:"targetDeviceSerial"`
	Username           string `json:"username" binding:"required"`
	Password           string `json:"password" binding:"required"`
}

// setStorageLocation godoc
// @Summary Change vault storage location
// @Description Migrates vault data to a different device. Requires vault unlocked and admin auth.
// @Tags vault
// @Accept json
// @Produce json
// @Param body body setStorageLocationRequest true "Migration request"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 423 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /vault/storage-location [put]
var setStorageLocationRoute = serverutil.ApiRoute(
	"PUT", "/vault/storage-location", func(c *gin.Context) *serverutil.Response {
		deps, _, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}

		var req setStorageLocationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(err)
		}

		ctx := c.Request.Context()

		if sessionUser, ok := ctxutil.Get[string](c, "username"); ok && sessionUser != req.Username {
			return serverutil.Unauthorized(fmt.Errorf("username does not match session"))
		}
		if _, err := authutil.ValidateBasicAuth(ctx, deps.Database().Queries, req.Username, req.Password); err != nil {
			return serverutil.Unauthorized(fmt.Errorf("invalid credentials"))
		}

		currentSerial, err := deps.Database().Queries.GetVaultLocation(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get current vault location: %w", err))
		}

		if currentSerial == req.TargetDeviceSerial {
			return serverutil.BadRequest(fmt.Errorf("vault is already on this device"))
		}

		var targetDB *db.DatabaseSqlc
		if req.TargetDeviceSerial == "" {
			targetDB = deps.Database()
		} else {
			var errResp *serverutil.Response
			targetDB, errResp = openVaultDBForSerial(deps, req.TargetDeviceSerial)
			if errResp != nil {
				return errResp
			}
		}

		sourceDB := deps.VaultDB()
		if err := backup.MigrateVault(ctx, sourceDB, targetDB); err != nil {
			if req.TargetDeviceSerial != "" {
				targetDB.Db.Close()
			}
			return serverutil.InternalServerError(fmt.Errorf("migrate vault: %w", err))
		}

		if err := backup.TruncateVaultTables(ctx, sourceDB); err != nil {
			if req.TargetDeviceSerial != "" {
				targetDB.Db.Close()
			}
			return serverutil.InternalServerError(fmt.Errorf("truncate source: %w", err))
		}

		if err := deps.Database().Queries.SetVaultLocation(ctx, req.TargetDeviceSerial); err != nil {
			if req.TargetDeviceSerial != "" {
				targetDB.Db.Close()
			}
			return serverutil.InternalServerError(fmt.Errorf("update vault location: %w", err))
		}

		if req.TargetDeviceSerial == "" {
			deps.ClearVaultDB()
		} else {
			deps.SetVaultDB(targetDB)
		}

		return serverutil.Ok().WithData(gin.H{
			"deviceSerial": req.TargetDeviceSerial,
			"migrated":     true,
		})
	},
)

func openVaultDBForSerial(deps deputil.Dependencies, serial string) (*db.DatabaseSqlc, *serverutil.Response) {
	if serial == "" {
		return nil, nil
	}

	device, err := deps.StorageService().FindManagedDeviceBySerial(serial)
	if err != nil {
		return nil, serverutil.InternalServerError(fmt.Errorf("find device: %w", err))
	}
	if device == nil {
		return nil, serverutil.BadRequest(fmt.Errorf("device %q not found or not connected", serial))
	}

	dbPath := filepath.Join(device.DataDir, "vault.db")
	vaultDB, err := db.ConnectToVaultDatabase(dbPath)
	if err != nil {
		return nil, serverutil.InternalServerError(fmt.Errorf("open vault db on device: %w", err))
	}

	return vaultDB, nil
}
