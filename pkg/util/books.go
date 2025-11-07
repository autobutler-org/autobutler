package util

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// FilterBookFiles filters a list of files to only include book files (PDF and EPUB)
func FilterBookFiles(files []fs.FileInfo) []fs.FileInfo {
	bookFiles := make([]fs.FileInfo, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileType := DetermineFileTypeFromPath(file.Name())
		if fileType == FileTypePDF || fileType == FileTypeEpub {
			bookFiles = append(bookFiles, file)
		}
	}
	return bookFiles
}

// RecursiveBookInfo stores a book with its relative path
type RecursiveBookInfo struct {
	FileInfo fs.FileInfo
	RelPath  string
}

// FindAllBooksRecursively finds all book files (PDF and EPUB) in a directory and its subdirectories
func FindAllBooksRecursively(rootDir string) ([]RecursiveBookInfo, error) {
	books := make([]RecursiveBookInfo, 0)

	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileType := DetermineFileTypeFromPath(info.Name())
		if fileType == FileTypePDF || fileType == FileTypeEpub {
			// Get relative path from rootDir
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			books = append(books, RecursiveBookInfo{
				FileInfo: info,
				RelPath:  relPath,
			})
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", rootDir, err)
	}

	return books, nil
}

