package bookutil

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

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

		fileType := storageutil.DetermineFileTypeFromPath(info.Name())
		if fileType == storageutil.FileTypePDF || fileType == storageutil.FileTypeEpub {
			// Get relative path from rootDir
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err // coverage: ignore - filepath.Rel only fails on cross-volume paths (different drives on Windows)
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
