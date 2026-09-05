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
// @Description Wipes the selected aspects of the appliance and logs the caller out everywhere. Both aspects are opt-in and a request selecting neither is rejected, so a truncated call cannot destroy anything. The confirm parameter must equal the authenticated username. The database is dropped and re-migrated in place, so no restart is required. Repeat calls are idempotent. Does not touch the health database, mount points, or external devices.
// @Tags auth
// @Produce json
// @Param database query bool false "Delete the Quark database"
// @Param files query bool false "Delete stored files"
// @Param confirm query string true "Must equal the authenticated username"
// @Success 200 {object} object{deleted=object{database=bool,files=bool}}
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
	if !deleteDatabase && !deleteFiles {
		return serverutil.BadRequest(fmt.Errorf("nothing selected: pass database=true, files=true, or both"))
	}
	if c.Query("confirm") != username {
		return serverutil.BadRequest(fmt.Errorf("confirm must be the authenticated username"))
	}

	database := (*deps).Database()
	user, err := database.Queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("failed to resolve authenticated user: %w", err))
	}

	result, err := authutil.DeleteAccount(c.Request.Context(), authutil.DeleteAccountParams{
		Database:       database,
		Queries:        database.Queries,
		DataDir:        storageutil.GetDataDir(),
		Username:       username,
		UserID:         user.ID,
		DeleteDatabase: deleteDatabase,
		DeleteFiles:    deleteFiles,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	clearSessionCookie(c)
	return serverutil.Ok().WithData(gin.H{"deleted": gin.H{
		"database": result.DatabaseDeleted,
		"files":    result.FilesDeleted,
	}})
}

var deleteAccountRoute = serverutil.ApiRoute("DELETE", "/auth/account", deleteAccount)
