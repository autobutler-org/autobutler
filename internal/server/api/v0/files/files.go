package v0_files

import (
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

// FileNodeJSON is a JSON-serializable representation of a file node. The
// listings build it in fileutil; the alias is what the endpoint annotations
// name it by.
type FileNodeJSON = fileutil.FileNode

// FileNodeWithTimeJSON extends FileNodeJSON with modification time for sorting/display
type FileNodeWithTimeJSON = fileutil.FileNodeWithTime

func NewRouter() serverutil.Router {
	return &router{}
}
