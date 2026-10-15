package fileutil_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/mholt/archiver/v4"
)

// compressed compresses content with one of archiver's compressors, the same
// library that reads it back.
func compressed(t *testing.T, c archiver.Compressor, content string) string {
	t.Helper()
	var buf bytes.Buffer
	w, err := c.OpenWriter(&buf)
	if err == nil {
		_, err = w.Write([]byte(content))
	}
	if err == nil {
		err = w.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// Every container added for #2990 lists, reads and extracts like the formats
// before it: compressed tars under their long and short names, comic book
// archives, and the packages that are zips underneath.
func TestMoreContainersListReadAndExtract(t *testing.T) {
	sevenZip, err := os.ReadFile(filepath.Join("testdata", "archive", "bundle.7z"))
	if err != nil {
		t.Fatal(err)
	}
	tarball := func(t *testing.T) string { return tarWith(t, archiveEntries) }
	formats := map[string]func(t *testing.T) string{
		"bundle.tar.zst": func(t *testing.T) string { return compressed(t, archiver.Zstd{}, tarball(t)) },
		"bundle.tzst":    func(t *testing.T) string { return compressed(t, archiver.Zstd{}, tarball(t)) },
		"bundle.tar.xz":  func(t *testing.T) string { return compressed(t, archiver.Xz{}, tarball(t)) },
		"bundle.txz":     func(t *testing.T) string { return compressed(t, archiver.Xz{}, tarball(t)) },
		"bundle.tar.bz2": func(t *testing.T) string { return compressed(t, archiver.Bz2{}, tarball(t)) },
		"bundle.tbz2":    func(t *testing.T) string { return compressed(t, archiver.Bz2{}, tarball(t)) },
		"bundle.cbz":     func(t *testing.T) string { return zipWith(t, archiveEntries) },
		"bundle.cb7":     func(*testing.T) string { return string(sevenZip) },
		"bundle.jar":     func(t *testing.T) string { return zipWith(t, archiveEntries) },
		"bundle.apk":     func(t *testing.T) string { return zipWith(t, archiveEntries) },
		"bundle.whl":     func(t *testing.T) string { return zipWith(t, archiveEntries) },
	}
	for name, build := range formats {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			registry, fsys, _ := archiveRegistry(t, archiveTargets[1])
			writeMem(t, fsys, name, build(t))

			listed, err := fileutil.ListArchive(fileutil.ListArchiveParams{
				Ctx: ctx, Registry: registry, FilePath: name, Serial: archiveSerial,
			})
			if err != nil {
				t.Fatalf("ListArchive failed: %v", err)
			}
			if got := nodeNames(listed.Entries); !slices.Equal(got, []string{"nested/", "top.txt"}) {
				t.Errorf("listed %v, want [nested/ top.txt]", got)
			}
			if got := readEntry(t, registry, archiveSerial, name, "nested/inner.txt"); got != "inner" {
				t.Errorf("nested/inner.txt = %q, want inner", got)
			}
			found, err := fileutil.FindArchive(fileutil.FindArchiveParams{
				Ctx: ctx, Registry: registry, FilePath: name + "/top.txt", Serial: archiveSerial,
			})
			if err != nil || !found.Found || found.ArchivePath != name || found.EntryPath != "top.txt" {
				t.Errorf("FindArchive(%s/top.txt) = %+v, %v; want the entry top.txt of %s", name, found, err, name)
			}

			extracted, err := fileutil.ExtractArchive(fileutil.ExtractArchiveParams{
				Ctx: ctx, Registry: registry, FilePath: name, Serial: archiveSerial,
			})
			if err != nil {
				t.Fatalf("ExtractArchive failed: %v", err)
			}
			if extracted.CreatedPath != "bundle" {
				t.Errorf("extracted into %q, want bundle", extracted.CreatedPath)
			}
			if got := readMem(t, fsys, "bundle/nested/inner.txt"); got != "inner" {
				t.Errorf("bundle/nested/inner.txt = %q, want inner", got)
			}
		})
	}
}

// A bare .zst, .xz or .bz2 is one compressed file, like a bare .gz: it lists
// as that file and decompresses beside itself.
func TestBareCompressedStreamsAreOneFile(t *testing.T) {
	formats := map[string]archiver.Compressor{
		"notes.txt.zst": archiver.Zstd{},
		"notes.txt.xz":  archiver.Xz{},
		"notes.txt.bz2": archiver.Bz2{},
	}
	for name, c := range formats {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			registry, fsys, _ := archiveRegistry(t, archiveTargets[1])
			writeMem(t, fsys, name, compressed(t, c, "hello"))

			listed, err := fileutil.ListArchive(fileutil.ListArchiveParams{
				Ctx: ctx, Registry: registry, FilePath: name, Serial: archiveSerial,
			})
			if err != nil {
				t.Fatalf("ListArchive failed: %v", err)
			}
			if got := nodeNames(listed.Entries); !slices.Equal(got, []string{"notes.txt"}) {
				t.Errorf("listed %v, want [notes.txt]", got)
			}
			if got := readEntry(t, registry, archiveSerial, name, "notes.txt"); got != "hello" {
				t.Errorf("notes.txt = %q, want hello", got)
			}
			extracted, err := fileutil.ExtractArchive(fileutil.ExtractArchiveParams{
				Ctx: ctx, Registry: registry, FilePath: name, Serial: archiveSerial,
			})
			if err != nil {
				t.Fatalf("ExtractArchive failed: %v", err)
			}
			if got := readMem(t, fsys, extracted.CreatedPath); extracted.CreatedPath != "notes.txt" || got != "hello" {
				t.Errorf("decompressed to %q holding %q, want notes.txt holding hello", extracted.CreatedPath, got)
			}
		})
	}
}

// The new containers obey the same limits: a jar is refused for a name that
// climbs out exactly as a .zip is.
func TestMoreContainersKeepTheSafetyLimits(t *testing.T) {
	ops := newArchiveOps(t, "evil.jar", zipWith(t, map[string]string{"../evil.class": "x"}), fileutil.ArchiveLimits{})
	var unsafe *fileutil.UnsafeArchiveEntryError
	if err := ops.list(); !errors.As(err, &unsafe) {
		t.Errorf("listing a jar with an escaping name: got %v, want an UnsafeArchiveEntryError", err)
	}
	if !storageutil.IsZipArchive("evil.jar") {
		t.Error("a jar should be read as a zip")
	}
}
