// Package v0_albums serves /api/v0/albums: the caller's photo albums, as a list or a nested tree, and the photos in
// them.
package v0_albums

import "github.com/autobutler-org/quark/pkg/util/serverutil"

func NewRouter() serverutil.Router {
	return &router{}
}
