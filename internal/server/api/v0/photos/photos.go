package v0_photos

import "github.com/autobutler-org/quark/pkg/util/serverutil"

// Router for /api/v0/photos endpoints
// Registers the /photos route

func NewRouter() serverutil.Router {
	return &router{}
}
