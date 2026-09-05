package xlsxutil

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
)

// openPart opens one part of the package for reading, bounded.
//
// A zip entry declares its own uncompressed size, and a small compressed part
// can expand to an arbitrarily large one, so neither the declaration nor the
// stream is trusted: the declared size rejects an honest oversized part
// outright, and the reader refuses to hand over more than the limit however
// small the part claimed to be. This is the guard the archive extractor
// already applies per entry (storageutil.MaxArchiveEntryBytes); a workbook is
// a zip and needs the same one.
func openPart(f *zip.File, limit int64) (io.ReadCloser, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", ErrTooLarge, f.Name, limit)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrNotSpreadsheet, f.Name, err)
	}
	return &boundedPart{
		reader: io.LimitReader(rc, limit+1),
		closer: rc,
		name:   f.Name,
		limit:  limit,
	}, nil
}

// boundedPart stops a part that expands past its limit, rather than letting it
// be read to the end of whatever it decompresses to.
type boundedPart struct {
	reader io.Reader
	closer io.Closer
	name   string
	limit  int64
	read   int64
}

func (b *boundedPart) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += int64(n)
	if b.read > b.limit {
		return n, fmt.Errorf("%w: %s expands past %d bytes", ErrTooLarge, b.name, b.limit)
	}
	return n, err
}

func (b *boundedPart) Close() error { return b.closer.Close() }

// partError reports what went wrong reading a part. A part that broke its
// bound says so; anything else is a malformed package.
func partError(name string, err error) error {
	if errors.Is(err, ErrTooLarge) {
		return err
	}
	return fmt.Errorf("%w: %s: %v", ErrNotSpreadsheet, name, err)
}
