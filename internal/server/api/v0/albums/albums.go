package v0_albums

import "github.com/autobutler-org/quark/pkg/util/serverutil"

// Router for /api/v0/albums endpoints

func NewRouter() serverutil.Router {
	return &router{}
}
