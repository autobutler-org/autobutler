package v0_version

import "github.com/autobutler-org/quark/pkg/util/serverutil"

const org = "autobutler-org"

func NewRouter() serverutil.Router {
	return &router{}
}
