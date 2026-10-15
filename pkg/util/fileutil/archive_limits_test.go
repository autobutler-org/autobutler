package fileutil_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"hash/crc32"
	"io"
	"slices"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// archiveOps runs each archive operation Quark offers against one archive, so
// a limit is shown to hold for listing, reading and extracting alike.
type archiveOps struct {
	registry vfs.Registry
	fsys     vfs.VFS
	path     string
	limits   fileutil.ArchiveLimits
}

func newArchiveOps(t *testing.T, name, archive string, limits fileutil.ArchiveLimits) archiveOps {
	t.Helper()
	registry, fsys, _ := archiveRegistry(t, archiveTargets[1])
	writeMem(t, fsys, name, archive)
	return archiveOps{registry: registry, fsys: fsys, path: name, limits: limits}
}

func (o archiveOps) list() error {
	_, err := fileutil.ListArchive(fileutil.ListArchiveParams{
		Ctx: context.Background(), Registry: o.registry, Limits: o.limits, FilePath: o.path, Serial: archiveSerial,
	})
	return err
}

// read opens entry and reads it to the end, which is where a stream that
// declared nothing finds out it is too big.
func (o archiveOps) read(entry string) error {
	opened, err := fileutil.OpenArchiveEntry(fileutil.OpenArchiveEntryParams{
		Ctx: context.Background(), Registry: o.registry, Limits: o.limits,
		ArchivePath: o.path, EntryPath: entry, Serial: archiveSerial,
	})
	if err != nil {
		return err
	}
	defer opened.Reader.Close()
	_, err = io.Copy(io.Discard, opened.Reader)
	return err
}

// extract extracts the archive and fails the test if a refused extraction left
// anything beside it.
func (o archiveOps) extract(t *testing.T) error {
	t.Helper()
	_, err := fileutil.ExtractArchive(fileutil.ExtractArchiveParams{
		Ctx: context.Background(), Registry: o.registry, Limits: o.limits, FilePath: o.path, Serial: archiveSerial,
	})
	if err != nil {
		if got := listNames(t, o.fsys, ""); !slices.Equal(got, []string{o.path}) {
			t.Errorf("after a refused extraction the folder holds %v, want only %s", got, o.path)
		}
	}
	return err
}

// wantLimit fails unless err is an ArchiveLimitError for limit, inside the
// UnsupportedError the handler answers with 400.
func wantLimit(t *testing.T, op string, err error, limit fileutil.ArchiveLimit) {
	t.Helper()
	var unsupported *fileutil.UnsupportedError
	var limitErr *fileutil.ArchiveLimitError
	if !errors.As(err, &unsupported) || !errors.As(err, &limitErr) {
		t.Fatalf("%s: got %T %v, want an ArchiveLimitError inside an UnsupportedError", op, err, err)
	}
	if limitErr.Limit != limit {
		t.Errorf("%s: tripped the %s limit, want %s", op, limitErr.Limit, limit)
	}
}

// zeroZip builds a zip of entries, each size zero bytes deflated: a
// megabyte of zeros is about a kilobyte stored, the shape of a zip bomb.
func zeroZip(t *testing.T, size int64, names ...string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err == nil {
			_, err = io.Copy(w, io.LimitReader(zeroReader{}, size))
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// rawZip builds a one-entry stored zip whose header says whatever h says,
// around content.
func rawZip(t *testing.T, h zip.FileHeader, content string) string {
	t.Helper()
	h.CRC32 = crc32.ChecksumIEEE([]byte(content))
	h.CompressedSize64 = uint64(len(content))
	if h.UncompressedSize64 == 0 {
		h.UncompressedSize64 = uint64(len(content))
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateRaw(&h)
	if err == nil {
		_, err = w.Write([]byte(content))
	}
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// tarOf builds an uncompressed tar of regular files, in the order given.
func tarOf(t *testing.T, entries ...[2]string) string {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		err := tw.WriteHeader(&tar.Header{Name: e[0], Typeflag: tar.TypeReg, Size: int64(len(e[1])), Mode: 0o644})
		if err == nil {
			_, err = tw.Write([]byte(e[1]))
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// One entry more than MaxEntries refuses the archive for every operation.
func TestArchiveRefusesTooManyEntries(t *testing.T) {
	ops := newArchiveOps(t, "many.zip", zipWith(t, map[string]string{"a": "1", "b": "2", "c": "3"}),
		fileutil.ArchiveLimits{MaxEntries: 2})
	wantLimit(t, "list", ops.list(), fileutil.ArchiveLimitEntries)
	wantLimit(t, "extract", ops.extract(t), fileutil.ArchiveLimitEntries)

	ops.limits.MaxEntries = 3
	if err := ops.list(); err != nil {
		t.Errorf("an archive at the entry limit should list: %v", err)
	}
}

// A zip bomb: a kilobyte that deflates to a megabyte. Its header is honest,
// so reading or extracting the entry is refused before a byte is inflated,
// while listing it, which inflates nothing, still works.
func TestArchiveRefusesAnEntryOverItsSize(t *testing.T) {
	bomb := zeroZip(t, 1<<20, "zeros.bin")
	if len(bomb) > 8<<10 {
		t.Fatalf("the fixture is %d bytes; it should be a small file that inflates", len(bomb))
	}
	ops := newArchiveOps(t, "bomb.zip", bomb, fileutil.ArchiveLimits{MaxEntryBytes: 64 << 10})
	if err := ops.list(); err != nil {
		t.Errorf("listing reads headers only and should work: %v", err)
	}
	wantLimit(t, "read", ops.read("zeros.bin"), fileutil.ArchiveLimitEntryBytes)
	wantLimit(t, "extract", ops.extract(t), fileutil.ArchiveLimitEntryBytes)
}

// A bare gzip says nothing of its size, so the cap is only found by reading:
// the served stream stops at it, and so does the extraction.
func TestArchiveCapsAStreamThatDeclaresNothing(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := io.Copy(gw, io.LimitReader(zeroReader{}, 1<<20)); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	ops := newArchiveOps(t, "zeros.gz", buf.String(), fileutil.ArchiveLimits{MaxEntryBytes: 64 << 10})
	wantLimit(t, "read", ops.read("zeros"), fileutil.ArchiveLimitEntryBytes)
	wantLimit(t, "extract", ops.extract(t), fileutil.ArchiveLimitEntryBytes)

	// Under the entry limit but over the total, the extraction is still
	// refused: the total counts what is actually written.
	ops.limits = fileutil.ArchiveLimits{MaxTotalBytes: 64 << 10}
	wantLimit(t, "extract", ops.extract(t), fileutil.ArchiveLimitTotalBytes)
}

// Entries that each pass but between them declare more than MaxTotalBytes
// refuse the archive, listing included: the shape of 42.zip, or of a bomb
// whose entries all point at the same compressed bytes.
func TestArchiveRefusesMoreThanItsTotal(t *testing.T) {
	cases := map[string]string{
		"bomb.zip":    zeroZip(t, 40<<10, "a.bin", "b.bin"),
		"bomb.tar.gz": gzipped(t, tarOf(t, [2]string{"a.bin", string(make([]byte, 40<<10))}, [2]string{"b.bin", string(make([]byte, 40<<10))})),
	}
	for name, archive := range cases {
		t.Run(name, func(t *testing.T) {
			ops := newArchiveOps(t, name, archive, fileutil.ArchiveLimits{MaxTotalBytes: 64 << 10})
			wantLimit(t, "list", ops.list(), fileutil.ArchiveLimitTotalBytes)
			wantLimit(t, "extract", ops.extract(t), fileutil.ArchiveLimitTotalBytes)
		})
	}
}

// A header claiming more than an int64 holds is not taken for an unknown
// size: the default limits refuse it, with nothing lowered for the test.
func TestArchiveRefusesAnImpossibleDeclaredSize(t *testing.T) {
	archive := rawZip(t, zip.FileHeader{Name: "huge.bin", Method: zip.Store, UncompressedSize64: 1<<64 - 1}, "tiny")
	ops := newArchiveOps(t, "huge.zip", archive, fileutil.ArchiveLimits{})
	wantLimit(t, "list", ops.list(), fileutil.ArchiveLimitTotalBytes)
}

// An entry nested deeper than MaxDepth refuses the archive.
func TestArchiveRefusesTooDeepANesting(t *testing.T) {
	archive := zipWith(t, map[string]string{"a/b/c/d.txt": "deep"})
	ops := newArchiveOps(t, "deep.zip", archive, fileutil.ArchiveLimits{MaxDepth: 3})
	wantLimit(t, "list", ops.list(), fileutil.ArchiveLimitDepth)
	wantLimit(t, "read", ops.read("a/b/c/d.txt"), fileutil.ArchiveLimitDepth)
	wantLimit(t, "extract", ops.extract(t), fileutil.ArchiveLimitDepth)

	ops.limits.MaxDepth = 4
	if err := ops.list(); err != nil {
		t.Errorf("an entry at the depth limit should list: %v", err)
	}
}

// A name that climbs out of the archive refuses the whole archive, so nothing
// lands beside the extraction folder (Zip Slip), whichever separator it uses.
func TestArchiveRefusesEscapingNames(t *testing.T) {
	cases := map[string]string{
		"parent.zip":    zipWith(t, map[string]string{"../evil.txt": "evil", "ok.txt": "ok"}),
		"inner.zip":     zipWith(t, map[string]string{"a/../../evil.txt": "evil"}),
		"backslash.zip": zipWith(t, map[string]string{`..\evil.txt`: "evil"}),
		// A tar streams, so a read checks only the entries before the one
		// it serves; the escaping one comes first here.
		"parent.tar": tarOf(t, [2]string{"../evil.txt", "evil"}, [2]string{"ok.txt", "ok"}),
	}
	for name, archive := range cases {
		t.Run(name, func(t *testing.T) {
			ops := newArchiveOps(t, name, archive, fileutil.ArchiveLimits{})
			for op, err := range map[string]error{"list": ops.list(), "read": ops.read("ok.txt"), "extract": ops.extract(t)} {
				var unsupported *fileutil.UnsupportedError
				var unsafe *fileutil.UnsafeArchiveEntryError
				if !errors.As(err, &unsupported) || !errors.As(err, &unsafe) {
					t.Errorf("%s: got %T %v, want an UnsafeArchiveEntryError inside an UnsupportedError", op, err, err)
				}
			}
		})
	}
}

// An encrypted entry still lists, since its name is not encrypted, but
// reading or extracting it is a typed error rather than ciphertext.
func TestArchiveReportsEncryptedEntries(t *testing.T) {
	archive := rawZip(t, zip.FileHeader{Name: "secret.txt", Method: zip.Store, Flags: 0x1}, "ciphertext")
	ops := newArchiveOps(t, "secret.zip", archive, fileutil.ArchiveLimits{})
	if err := ops.list(); err != nil {
		t.Errorf("an encrypted archive's names should still list: %v", err)
	}
	for op, err := range map[string]error{"read": ops.read("secret.txt"), "extract": ops.extract(t)} {
		var unsupported *fileutil.UnsupportedError
		var encrypted *fileutil.EncryptedArchiveError
		if !errors.As(err, &unsupported) || !errors.As(err, &encrypted) {
			t.Errorf("%s: got %T %v, want an EncryptedArchiveError inside an UnsupportedError", op, err, err)
			continue
		}
		if encrypted.Name != "secret.txt" {
			t.Errorf("%s: names %q, want the entry secret.txt", op, encrypted.Name)
		}
	}
}
