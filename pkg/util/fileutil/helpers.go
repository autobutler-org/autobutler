package fileutil

import (
	"archive/zip"
	"fmt"
	"io"
)

// notFound marks an error as a 404 for the handler.
func notFound(err error) error { return &NotFoundError{Err: err} }

// notFoundf marks a formatted message as a 404 for the handler.
func notFoundf(format string, args ...any) error {
	return &NotFoundError{Err: fmt.Errorf(format, args...)}
}

// zipMethodNames names the compression methods a zip may declare. Go's
// archive/zip implements only Store (0) and Deflate (8); everything else comes
// back as the bare "zip: unsupported compression algorithm", which tells the
// user nothing about which archive is at fault or what to do about it.
var zipMethodNames = map[uint16]string{
	1:  "Shrink",
	2:  "Reduce",
	6:  "Implode",
	9:  "Deflate64",
	12: "BZIP2",
	14: "LZMA",
	93: "Zstandard",
	95: "XZ",
	96: "JPEG",
	97: "WavPack",
	98: "PPMd",
}

// openZipEntry opens an entry, translating a method this build cannot
// decompress into an [UnsupportedError] that names the method and the fix.
func openZipEntry(f *zip.File) (io.ReadCloser, error) {
	rc, err := f.Open()
	if err == nil {
		return rc, nil
	}
	if f.Method != zip.Store && f.Method != zip.Deflate {
		name, known := zipMethodNames[f.Method]
		if !known {
			name = fmt.Sprintf("method %d", f.Method)
		}
		return nil, &UnsupportedError{Err: fmt.Errorf(
			"archive entry %s uses %s compression, which is not supported; re-create the archive using standard Deflate compression",
			f.Name, name)}
	}
	return nil, fmt.Errorf("failed to open archive entry %s: %w", f.Name, err)
}
