package v0_smb

import "github.com/autobutler-org/quark/pkg/util/serverutil"

// SmbStatusJSON is the response for GET /smb/status.
type SmbStatusJSON struct {
	Linux      bool   `json:"linux"`
	Installed  bool   `json:"installed"`
	Configured bool   `json:"configured"`
	Running    bool   `json:"running"`
	FilesDir   string `json:"filesDir"`
}

func NewRouter() serverutil.Router {
	return &router{}
}
