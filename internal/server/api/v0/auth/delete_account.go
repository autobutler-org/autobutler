package v0_auth

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// deleteAccount godoc
// @Summary Delete account data (factory reset)
// @Description Wipes the selected aspects of the appliance and logs the caller out everywhere. All three aspects are opt-in and a request selecting none is rejected, so a truncated call cannot destroy anything. The confirm parameter must equal the authenticated username. Databases are dropped and re-migrated in place, so no restart is required. Repeat calls are idempotent. External device data is reached only when devices=true; a drive that is not attached at reset time keeps its data.
// @Tags auth
// @Produce json
// @Param database query bool false "Delete the appliance databases (quark.db, quark.health.db)"
// @Param files query bool false "Delete stored files under the data directory"
// @Param devices query bool false "Delete the Quark data directory on attached external devices"
// @Param confirm query string true "Must equal the authenticated username"
// @Success 200 {object} object{deleted=object{database=bool,files=bool,devices=bool}}
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Router /auth/account [delete]
func deleteAccount(c *gin.Context) *serverutil.Response {
	deps, ok := getQueries(c)
	if !ok || (*deps).Database() == nil {
		return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
	}

	// requireAuth sets "username", not "userID" — the user is resolved from it
	// below rather than read straight out of the context.
	username, ok := ctxutil.Get[string](c, "username")
	if !ok || username == "" {
		return serverutil.Unauthorized(fmt.Errorf("not authenticated"))
	}

	deleteDatabase, ok := queryBool(c, "database")
	if !ok {
		return serverutil.BadRequest(fmt.Errorf("database must be a boolean"))
	}
	deleteFiles, ok := queryBool(c, "files")
	if !ok {
		return serverutil.BadRequest(fmt.Errorf("files must be a boolean"))
	}
	deleteDevices, ok := queryBool(c, "devices")
	if !ok {
		return serverutil.BadRequest(fmt.Errorf("devices must be a boolean"))
	}
	if !deleteDatabase && !deleteFiles && !deleteDevices {
		return serverutil.BadRequest(fmt.Errorf("nothing selected: pass database=true, files=true, devices=true, or any combination"))
	}
	if c.Query("confirm") != username {
		return serverutil.BadRequest(fmt.Errorf("confirm must be the authenticated username"))
	}

	database := (*deps).Database()
	user, err := database.Queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to resolve authenticated user: %w", err))
	}

	// Resolved only when devices are actually in scope: listing managed devices
	// probes the attached drives, and nothing should touch them otherwise.
	var deviceDataDirs []string
	if deleteDevices {
		deviceDataDirs, err = externalDeviceDataDirs(*deps)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to list external devices: %w", err))
		}
		// An off-appliance vault lives inside one of those directories, so its
		// handle has to be closed before the directory goes or the process
		// keeps writing to an unlinked file.
		(*deps).ClearVaultDB()
	}

	result, err := authutil.DeleteAccount(c.Request.Context(), authutil.DeleteAccountParams{
		Database:       database,
		Queries:        database.Queries,
		HealthDatabase: (*deps).HealthDatabase(),
		DataDir:        storageutil.GetDataDir(),
		DeviceDataDirs: deviceDataDirs,
		Username:       username,
		UserID:         user.ID,
		DeleteDatabase: deleteDatabase,
		DeleteFiles:    deleteFiles,
		DeleteDevices:  deleteDevices,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	clearSessionCookie(c)
	return serverutil.Ok().WithData(gin.H{"deleted": gin.H{
		"database": result.DatabaseDeleted,
		"files":    result.FilesDeleted,
		"devices":  result.DevicesDeleted,
	}})
}

var deleteAccountRoute = serverutil.ApiRoute("DELETE", "/auth/account", deleteAccount)
