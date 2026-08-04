// Package v0_admin implements admin role management endpoints (#1204).
//
// Role hierarchy:
//
//	owner → admin → user
//
// 'owner' is the initial account created at setup. There is always exactly one
// owner. Both 'owner' and 'admin' are considered admin-privileged; only 'user'
// is restricted.
//
// Routes (all require admin privilege except GET /admin/users):
//
//	GET  /admin/users           — list all users and their roles
//	POST /admin/users/:id/promote — promote a user to admin
//	POST /admin/users/:id/demote  — demote an admin to user (blocked if last admin)
package v0_admin

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// requireAdmin is a helper that returns the caller's userID and verifies they
// have admin or owner role. Returns a *serverutil.Response error when the
// check fails (caller should return it immediately).
func requireAdmin(c *gin.Context, deps deputil.Dependencies) (int64, *serverutil.Response) {
	userID, ok := ctxutil.Get[int64](c, "userID")
	if !ok {
		return 0, serverutil.Unauthorized(errors.New("not authenticated"))
	}
	isAdmin, err := deps.Database().Queries.IsAdmin(c.Request.Context(), userID)
	if err != nil {
		return 0, serverutil.InternalServerError(err)
	}
	if isAdmin != 1 {
		return 0, serverutil.Unauthorized(errors.New("admin privilege required"))
	}
	return userID, nil
}

// UserRoleJSON is the API representation of a user's role.
type UserRoleJSON struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// listUsers godoc
// @Summary List users and their roles
// @Description Returns all users with their assigned roles. Requires admin privilege.
// @Tags admin
// @Produce json
// @Success 200 {array} UserRoleJSON
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /admin/users [get]
func listUsers(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	if _, resp := requireAdmin(c, deps); resp != nil {
		return resp
	}

	rows, err := deps.Database().Queries.ListUserRoles(c.Request.Context())
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	result := make([]UserRoleJSON, len(rows))
	for i, r := range rows {
		result[i] = UserRoleJSON{
			ID:       r.ID,
			Username: r.Username,
			Role:     r.Role,
		}
	}
	return serverutil.Ok().WithData(result)
}

// promoteUser godoc
// @Summary Promote a user to admin
// @Description Grants admin privilege to the specified user. Requires admin privilege.
// @Tags admin
// @Param id path int true "User ID to promote"
// @Produce json
// @Success 200 {object} UserRoleJSON
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /admin/users/{id}/promote [post]
func promoteUser(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	if _, resp := requireAdmin(c, deps); resp != nil {
		return resp
	}

	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return serverutil.BadRequest(err)
	}

	ctx := c.Request.Context()
	currentRole, err := deps.Database().Queries.GetUserRole(ctx, targetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return serverutil.NotFound(fmt.Errorf("user %d not found", targetID))
		}
		return serverutil.InternalServerError(err)
	}
	if currentRole == "owner" {
		return serverutil.BadRequest(errors.New("cannot change the owner's role"))
	}
	if currentRole == "admin" {
		return serverutil.BadRequest(errors.New("user is already an admin"))
	}

	if err := deps.Database().Queries.SetUserRole(ctx, db.SetUserRoleParams{Role: "admin", ID: targetID}); err != nil {
		return serverutil.InternalServerError(err)
	}

	rows, err := deps.Database().Queries.ListUserRoles(ctx)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	for _, r := range rows {
		if r.ID == targetID {
			return serverutil.Ok().WithData(UserRoleJSON{ID: r.ID, Username: r.Username, Role: r.Role})
		}
	}
	return serverutil.InternalServerError(errors.New("user not found after update"))
}

// demoteUser godoc
// @Summary Demote an admin to user
// @Description Removes admin privilege from the specified user. Blocked if this would leave zero admins. Requires admin privilege.
// @Tags admin
// @Param id path int true "User ID to demote"
// @Produce json
// @Success 200 {object} UserRoleJSON
// @Failure 400 {object} serverutil.Response
// @Failure 401 {object} serverutil.Response
// @Failure 404 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /admin/users/{id}/demote [post]
func demoteUser(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	if _, resp := requireAdmin(c, deps); resp != nil {
		return resp
	}

	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return serverutil.BadRequest(err)
	}

	ctx := c.Request.Context()
	currentRole, err := deps.Database().Queries.GetUserRole(ctx, targetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return serverutil.NotFound(fmt.Errorf("user %d not found", targetID))
		}
		return serverutil.InternalServerError(err)
	}
	if currentRole == "owner" {
		return serverutil.BadRequest(errors.New("cannot demote the owner"))
	}
	if currentRole == "user" {
		return serverutil.BadRequest(errors.New("user is already a regular user"))
	}

	// Guard: prevent demoting the last admin.
	count, err := deps.Database().Queries.CountAdmins(ctx)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	if count <= 1 {
		return serverutil.BadRequest(errors.New("cannot demote the last admin — promote another user first"))
	}

	if err := deps.Database().Queries.SetUserRole(ctx, db.SetUserRoleParams{Role: "user", ID: targetID}); err != nil {
		return serverutil.InternalServerError(err)
	}

	rows, err := deps.Database().Queries.ListUserRoles(ctx)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	for _, r := range rows {
		if r.ID == targetID {
			return serverutil.Ok().WithData(UserRoleJSON{ID: r.ID, Username: r.Username, Role: r.Role})
		}
	}
	return serverutil.InternalServerError(errors.New("user not found after update"))
}

type router struct{}

// NewRouter returns the router for admin role management.
func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		serverutil.ApiRoute("GET", "/admin/users", listUsers),
		serverutil.ApiRoute("POST", "/admin/users/:id/promote", promoteUser),
		serverutil.ApiRoute("POST", "/admin/users/:id/demote", demoteUser),
	}
}
