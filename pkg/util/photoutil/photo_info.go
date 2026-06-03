package photoutil

import "io/fs"

// PhotoInfo stores a photo with its relative path
type PhotoInfo struct {
	FileInfo     fs.FileInfo
	RelPath      string
	HasLiveVideo bool
}
