package util

import (
	"fmt"
	"path/filepath"
)

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

func GetFileType(filename string) FileType {
	ext := filepath.Ext(filename)
	switch ext {
	case ".pdf":
		return FileTypePDF
	case ".pptx", ".ppt":
		return FileTypeSlideshow
	case ".png", ".jpg", ".jpeg", ".gif":
		return FileTypeImage
	case "":
		if filename[len(filename)-1] == '/' {
			return FileTypeFolder
		}
		return FileTypeGeneric
	default:
		return FileTypeGeneric
	}
}

func IsFileType(filename string, expected FileType) bool {
	actual := GetFileType(filename)
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
