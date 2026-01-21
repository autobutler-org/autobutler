package storageutil

import (
	"io/fs"
	"strings"
	"time"
)

// CustomFileInfo is a simple implementation of fs.FileInfo, allowing us to set the fields ourselves.
type CustomFileInfo struct {
	name string
	size int64
}

func (f CustomFileInfo) Name() string {
	return f.name
}
func (f CustomFileInfo) Size() int64 {
	return f.size
}
func (f CustomFileInfo) Mode() fs.FileMode {
	return 0666
}
func (f CustomFileInfo) ModTime() time.Time {
	return time.Now()
}
func (f CustomFileInfo) IsDir() bool {
	return f.name[len(f.name)-1] == '/'
}
func (f CustomFileInfo) Sys() any {
	return nil
}
func NewCustomFileInfo() *CustomFileInfo {
	return &CustomFileInfo{}
}
func (f *CustomFileInfo) WithName(name string) *CustomFileInfo {
	if !strings.HasSuffix(name, "/") {
		name = name + "/"
	}
	f.name = name
	return f
}
func (f *CustomFileInfo) WithSize(size int64) *CustomFileInfo {
	f.size = size
	return f
}
