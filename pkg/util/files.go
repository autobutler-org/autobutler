package util

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"time"
)

type TestFileInfo struct {
    name string
    size int64
}
func (f TestFileInfo) Name() string {
    return f.name
}
func (f TestFileInfo) Size() int64 {
    return f.size
}
func (f TestFileInfo) Mode() fs.FileMode {
    return 0666
}
func (f TestFileInfo) ModTime() time.Time {
    return time.Now()
}
func (f TestFileInfo) IsDir() bool {
    return f.name[len(f.name)-1] == '/'
}
func (f TestFileInfo) Sys() any {
    return nil
}
func NewTestFileInfo(name string, size int64) fs.FileInfo {
	return TestFileInfo{name: name, size: size}
}

type FileType string

const (
	FileTypeGeneric   FileType = "generic"
	FileTypePDF      FileType = "pdf"
	FileTypeSlideshow FileType = "slideshow"
	FileTypeImage     FileType = "image"
	FileTypeFolder    FileType = "folder"
)

func KB(size float64) int64 {
	return int64(size * 1024)
}

func MB(size float64) int64 {
	return int64(KB(size) * 1024)
}

func GB(size float64) int64 {
	return int64(MB(size) * 1024)
}

func TB(size float64) int64 {
	return int64(GB(size) * 1024)
}

func DetermineFileType(file fs.FileInfo) FileType {
	if file.IsDir() {
		return FileTypeFolder
	}
	ext := filepath.Ext(file.Name())
	switch ext {
	case ".pdf":
		return FileTypePDF
	case ".pptx", ".ppt":
		return FileTypeSlideshow
	case ".png", ".jpg", ".jpeg", ".gif":
		return FileTypeImage
	default:
		return FileTypeGeneric
	}
}

func IsFileType(file fs.FileInfo, expected FileType) bool {
	actual := DetermineFileType(file)
	return actual == expected
}

func SizeBytesToString(size_bytes int64) string {
    if size_bytes < 1024 {
        return fmt.Sprintf("%d B", size_bytes)
    } else if size_bytes < 1024*1024 {
        return fmt.Sprintf("%.1f KB", float64(size_bytes)/1024)
    } else if size_bytes < 1024*1024*1024 {
        return fmt.Sprintf("%.1f MB", float64(size_bytes)/(1024*1024))
    } else if size_bytes < 1024*1024*1024*1024 {
        return fmt.Sprintf("%.1f GB", float64(size_bytes)/(1024*1024*1024))
    } else {
        return fmt.Sprintf("%.1f TB", float64(size_bytes)/(1024*1024*1024*1024))
    }
}
