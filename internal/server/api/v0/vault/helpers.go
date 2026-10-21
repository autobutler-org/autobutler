package v0_vault

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

// errBodyTooLarge is the 413 a body over its limit gets.
var errBodyTooLarge = errors.New("request body is too large")

// callerContext is what every vault handler reads first: the dependencies,
// and the signed-in account the request acts on.
func callerContext(c *gin.Context) (deputil.Dependencies, int64, *serverutil.Response) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil {
		return deps, 0, serverutil.InternalServerError(errors.New("database unavailable"))
	}
	principal, _ := ctxutil.Get[accessutil.Principal](c, "principal")
	if principal.UserID == 0 {
		return deps, 0, serverutil.Unauthorized(errors.New("not authenticated"))
	}
	return deps, principal.UserID, nil
}

// reconfirm checks that password is the auth key of the caller's login
// password, so a session alone neither sets up nor wipes a vault. A wrong or
// empty key is a 403 and a raw password a 426; nil means it matched.
func reconfirm(c *gin.Context, deps deputil.Dependencies, password string) *serverutil.Response {
	username, _ := ctxutil.Get[string](c, "username")
	_, err := authutil.VerifyPassword(c.Request.Context(), authutil.VerifyPasswordParams{
		Queries:  deps.Database().Queries,
		Username: username,
		Password: password,
	})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, authutil.ErrIncorrectPassword):
		return serverutil.NewResponse().WithStatusCode(http.StatusForbidden).WithError(err)
	case errors.Is(err, authutil.ErrUserNotFound):
		return serverutil.Unauthorized(errors.New("not authenticated"))
	default:
		// ErrAppTooOld is a 426 HttpError, whatever status this names.
		return serverutil.InternalServerError(err)
	}
}

// itemID reads the :id path parameter. One that isn't a positive integer
// names no item.
func itemID(c *gin.Context) (int64, *serverutil.Response) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, serverutil.NotFound(vaultutil.ErrItemNotFound)
	}
	return id, nil
}

// bindBody decodes a JSON body of at most limit bytes into out. A body over
// the limit is a 413 and any other malformed body a 400 with invalid.
func bindBody(c *gin.Context, limit int64, out any, invalid error) *serverutil.Response {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	err := c.ShouldBindJSON(out)
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		return serverutil.NewResponse().WithStatusCode(http.StatusRequestEntityTooLarge).WithError(errBodyTooLarge)
	}
	if err != nil {
		return serverutil.BadRequest(invalid)
	}
	return nil
}

// vaultError maps an error from vaultutil to its status code. The sentinels'
// text is written for the app to show, so it goes out unwrapped.
func vaultError(err error) *serverutil.Response {
	switch {
	case errors.Is(err, vaultutil.ErrKeyNotFound), errors.Is(err, vaultutil.ErrItemNotFound):
		return serverutil.NotFound(err)
	case errors.Is(err, vaultutil.ErrKeyExists):
		return serverutil.Conflict(err)
	case errors.Is(err, vaultutil.ErrInvalidKey), errors.Is(err, vaultutil.ErrInvalidItem):
		return serverutil.BadRequest(err)
	default:
		return serverutil.InternalServerError(err)
	}
}
