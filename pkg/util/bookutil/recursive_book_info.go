package bookutil

import "io/fs"

// RecursiveBookInfo stores a book with its relative path
type RecursiveBookInfo struct {
	FileInfo fs.FileInfo
	RelPath  string
}
