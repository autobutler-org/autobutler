package photoutil_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/photoutil"
)

// countingReader reports how much of the source was pulled into memory.
type countingReader struct {
	r    io.Reader
	read int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += n
	return n, err
}

// seekableSource is what vfs.VFS.Open hands back in production: a reader that
// already seeks. AsReadSeeker must use it in place.
type seekableSource struct {
	*bytes.Reader
	reads int
}

func (s *seekableSource) Read(p []byte) (int, error) {
	n, err := s.Reader.Read(p)
	s.reads += n
	return n, err
}

// The whole point of AsReadSeeker: a source that already seeks is handed back
// untouched, so nothing is copied onto the heap. The thumbnail path used to
// io.ReadAll every image unconditionally (#1723).
func TestAsReadSeekerDoesNotBufferASeekableSource(t *testing.T) {
	src := &seekableSource{Reader: bytes.NewReader(make([]byte, 1<<20))}

	rs, err := photoutil.AsReadSeeker(src)
	if err != nil {
		t.Fatalf("AsReadSeeker: %v", err)
	}
	if rs != io.ReadSeeker(src) {
		t.Fatal("a seekable source must be returned as-is, not copied into a buffer")
	}
	if src.reads != 0 {
		t.Errorf("nothing should have been read yet, got %d bytes", src.reads)
	}
}

// A stream-only source still has to work — it is the fallback, and it is the
// only case where the content lands in memory.
func TestAsReadSeekerBuffersOnlyAStreamOnlySource(t *testing.T) {
	content := []byte("stream-only content")
	src := &countingReader{r: bytes.NewReader(content)}

	rs, err := photoutil.AsReadSeeker(src)
	if err != nil {
		t.Fatalf("AsReadSeeker: %v", err)
	}
	got, err := io.ReadAll(rs)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content: got %q, want %q", got, content)
	}
	// It must still seek, which is the reason the fallback exists at all.
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("the fallback must be seekable: %v", err)
	}
	if src.read != len(content) {
		t.Errorf("fallback should have read the whole source, got %d of %d", src.read, len(content))
	}
}
