package util

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

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
func NewCustomFileInfo(name string, size int64) fs.FileInfo {
	return CustomFileInfo{name: name, size: size}
}

type FileType string

const (
	FileTypeDocx      FileType = "docx"
	FileTypeEpub      FileType = "epub"
	FileTypeFolder    FileType = "folder"
	FileTypeGeneric   FileType = "generic"
	FileTypeImage     FileType = "image"
	FileTypePDF       FileType = "pdf"
	FileTypeSlideshow FileType = "slideshow"
	FileTypeVideo     FileType = "video"
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

func DetermineFileTypeFromPath(filePath string) FileType {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".pdf":
		return FileTypePDF
	case ".pptx", ".ppt":
		return FileTypeSlideshow
	case ".png", ".jpg", ".jpeg", ".gif":
		return FileTypeImage
	case ".mp4", ".m4v", ".webm", ".ogg", ".avi", ".mov":
		return FileTypeVideo
	case ".epub":
		return FileTypeEpub
	case ".docx":
		return FileTypeDocx
	case "/":
		return FileTypeFolder
	default:
		stat, err := os.Stat(filePath)
		if err == nil && stat != nil {
			if stat.IsDir() {
				return FileTypeFolder
			}
		}
		return FileTypeGeneric
	}
}

func DetermineFileType(rootDir string, file fs.FileInfo) FileType {
	if file.IsDir() {
		return FileTypeFolder
	}
	filesDir := GetFilesDir()
	stat, err := os.Stat(filepath.Join(filesDir, rootDir, file.Name()))
	if err != nil || stat == nil {
		return FileTypeGeneric // If we can't stat the file, treat it as generic
	}
	if stat.IsDir() {
		return FileTypeFolder
	}
	return DetermineFileTypeFromPath(file.Name())
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

func GetFolderSize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(_ string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error calculating folder size for %s: %w", dir, err)
	}
	return size, nil
}

func StatFilesInDir(dir string) ([]fs.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	files := make([]fs.FileInfo, len(entries))
	if err != nil {
		return nil, fmt.Errorf("error reading the directory %s: %w", dir, err)
	}
	for i, entry := range entries {
		if entry.IsDir() {
			folderSize, err := GetFolderSize(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("error getting size for folder %s: %w", entry.Name(), err)
			}
			files[i] = NewCustomFileInfo(entry.Name()+"/", folderSize)
		} else {
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("error getting info for file %s: %w", entry.Name(), err)
			}
			files[i] = info
		}
	}
	// Sort files by directory first, then by name
	slices.SortFunc(files, func(a, b fs.FileInfo) int {
		if a.IsDir() && !b.IsDir() {
			return -1 // a is a directory, b is a file
		} else if !a.IsDir() && b.IsDir() {
			return 1 // a is a file, b is a directory
		}
		return strings.Compare(a.Name(), b.Name())
	})
	return files, nil
}
