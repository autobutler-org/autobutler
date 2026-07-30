package v0_storage

import (
	"fmt"
	"path/filepath"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/backup"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

var validRoles = map[string]bool{
	"default-storage": true,
	"snapshot-backup": true,
	"unassigned":      true,
}

// setDeviceRole godoc
// @Summary Set the role of a storage device
// @Description Assigns a role (default-storage, snapshot-backup, unassigned) to a device. Requires master password re-entry. When setting role=default-storage, pass moveVault=true to also migrate the vault database to that device.
// @Tags storage
// @Accept json
// @Produce json
// @Param body body object true "{serial: string, role: string, username: string, password: string, moveVault?: bool}"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /storage/devices/role [put]
func setDeviceRole(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil {
		return serverutil.InternalServerError(nil)
	}

	var req struct {
		Serial    string `json:"serial"`
		Role      string `json:"role" binding:"required"`
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
		MoveVault bool   `json:"moveVault"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(err)
	}

	if !validRoles[req.Role] {
		return serverutil.BadRequest(fmt.Errorf("role must be one of: default-storage, snapshot-backup, unassigned"))
	}

	ctx := c.Request.Context()

	// Verify username matches the authenticated session.
	if sessionUser, ok := ctxutil.Get[string](c, "username"); ok && sessionUser != req.Username {
		return serverutil.Unauthorized(fmt.Errorf("username does not match session"))
	}

	// Re-validate master password.
	if _, err := authutil.ValidateBasicAuth(ctx, deps.Database().Queries, req.Username, req.Password); err != nil {
		return serverutil.Unauthorized(fmt.Errorf("invalid credentials"))
	}

	// For non-unassigned roles, verify the device is actually connected.
	if req.Role != "unassigned" {
		statuses, err := deps.StorageService().GetDeviceStatuses()
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to list devices: %w", err))
		}
		found := false
		isInternal := false
		for _, s := range statuses {
			if req.Serial == "" && s.IsInternal {
				found = true
				isInternal = true
				break
			}
			if s.UsbInfo != nil && s.UsbInfo.GetSerial() == req.Serial {
				found = true
				break
			}
		}
		if !found {
			return serverutil.BadRequest(fmt.Errorf("device with serial %q not found", req.Serial))
		}
		if isInternal && req.Role == "snapshot-backup" {
			return serverutil.BadRequest(fmt.Errorf("internal device cannot be assigned as snapshot-backup"))
		}
	}

	// If assigning default-storage, clear any existing default-storage first.
	if req.Role == "default-storage" {
		if err := deps.Database().Queries.ClearDefaultStorageRole(ctx); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to clear existing default-storage: %w", err))
		}
	}

	if err := deps.Database().Queries.UpsertDeviceRole(ctx, db.UpsertDeviceRoleParams{
		DeviceSerial: req.Serial,
		Role:         req.Role,
	}); err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to set device role: %w", err))
	}

	vaultMigrated := false

	// When setting a device as default-storage and the caller opts in to moving
	// the vault, migrate the vault database to the new primary device.
	if req.Role == "default-storage" && req.MoveVault && deps.VaultSession() != nil {
		currentSerial, err := deps.Database().Queries.GetVaultLocation(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get vault location: %w", err))
		}
		if currentSerial == req.Serial {
			// Already on this device — nothing to migrate.
			vaultMigrated = false
		} else {
			if _, unlocked := deps.VaultSession().Key(); !unlocked {
				return serverutil.NewResponse().
					WithStatusCode(423).
					WithError(fmt.Errorf("vault must be unlocked to move it"))
			}

			var targetDB *db.DatabaseSqlc
			if req.Serial == "" {
				// Moving back to internal storage.
				internalDataDir := storageutil.GetDataDir()
				dbPath := filepath.Join(internalDataDir, "vault.db")
				if targetDB, err = db.ConnectToVaultDatabase(dbPath); err != nil {
					return serverutil.InternalServerError(fmt.Errorf("open internal vault db: %w", err))
				}
			} else {
				device, err := deps.StorageService().FindManagedDeviceBySerial(req.Serial)
				if err != nil || device == nil {
					return serverutil.BadRequest(fmt.Errorf("device %q not found or not connected", req.Serial))
				}
				dbPath := filepath.Join(device.DataDir, "vault.db")
				if targetDB, err = db.ConnectToVaultDatabase(dbPath); err != nil {
					return serverutil.InternalServerError(fmt.Errorf("open vault db on device: %w", err))
				}
			}

			sourceDB := deps.VaultDB()
			if err := backup.MigrateVault(ctx, sourceDB, targetDB); err != nil {
				if req.Serial != "" {
					targetDB.Db.Close()
				}
				return serverutil.InternalServerError(fmt.Errorf("migrate vault: %w", err))
			}
			if err := backup.TruncateVaultTables(ctx, sourceDB); err != nil {
				if req.Serial != "" {
					targetDB.Db.Close()
				}
				return serverutil.InternalServerError(fmt.Errorf("truncate source vault: %w", err))
			}
			if err := deps.Database().Queries.SetVaultLocation(ctx, req.Serial); err != nil {
				if req.Serial != "" {
					targetDB.Db.Close()
				}
				return serverutil.InternalServerError(fmt.Errorf("update vault location: %w", err))
			}
			if req.Serial == "" {
				deps.ClearVaultDB()
			} else {
				deps.SetVaultDB(targetDB)
			}
			vaultMigrated = true
		}
	}

	return serverutil.Ok().WithData(gin.H{
		"serial":        req.Serial,
		"role":          req.Role,
		"vaultMigrated": vaultMigrated,
	})
}

var setDeviceRoleRoute = serverutil.ApiRoute(
	"PUT", "/storage/devices/role", setDeviceRole,
)
