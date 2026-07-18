package v0_auth

import "github.com/autobutler-org/autobutler/pkg/util/serverutil"

type router struct {
	// insecure mirrors the --insecure / AUTOBUTLER_INSECURE flag. When true the
	// server is running plain HTTP and the Secure cookie attribute must be omitted
	// (browsers reject Secure cookies over HTTP). When false (TLS active) we set
	// Secure: true so the cookie cannot be transmitted over plain HTTP.
	insecure bool
}

// NewRouter constructs the auth router. Pass insecure=true when the server
// is started without TLS so that session cookies do NOT carry the Secure flag
// (required for HTTP, rejected by browsers on HTTPS endpoints).
func NewRouter(insecure bool) serverutil.Router {
	return &router{insecure: insecure}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		// Local auth
		serverutil.ApiRoute("GET", "/auth/status", authStatus),
		serverutil.ApiRoute("POST", "/auth/setup", r.authSetup),
		serverutil.ApiRoute("POST", "/auth/login", r.authLogin),
		serverutil.ApiRoute("POST", "/auth/logout", r.authLogout),
		serverutil.ApiRoute("POST", "/auth/recover", r.authRecover),
		// Google OAuth (for cloud migration feature)
		googleAuthorizeRoute,
		googleCallbackRoute,
		googleDisconnectRoute,
	}
}
