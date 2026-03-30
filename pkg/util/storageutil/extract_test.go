package storageutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildZip creates an in-memory zip archive from the provided entries.
// Each entry is a (name, content) pair.
func buildZip(t *testing.T, entries []struct{ name, content string }) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		f, err := w.Create(e.name)
		if err != nil {
			t.Fatalf("buildZip: create entry %q: %v", e.name, err)
		}
		if _, err := f.Write([]byte(e.content)); err != nil {
			t.Fatalf("buildZip: write entry %q: %v", e.name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("buildZip: close: %v", err)
	}
	return buf.Bytes()
}

func writeZipToFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writeZipToFile: %v", err)
	}
	return path
}

func makeDevice(t *testing.T) *ManagedDevice {
	t.Helper()
	dir := t.TempDir()
	cirrus := filepath.Join(dir, "cirrus")
	if err := os.MkdirAll(cirrus, 0755); err != nil {
		t.Fatalf("makeDevice: %v", err)
	}
	return &ManagedDevice{
		Device:    Device{Name: "test", MountPoint: dir, IsInternal: true},
		DataDir:   dir,
		CirrusDir: cirrus,
	}
}

func TestExtractFileImpl_BasicExtraction(t *testing.T) {
	device := makeDevice(t)

	data := buildZip(t, []struct{ name, content string }{
		{"hello.txt", "hello world"},
		{"subdir/nested.txt", "nested content"},
	})
	writeZipToFile(t, device.CirrusDir, "archive.zip", data)

	result, err := ExtractFileImpl(ExtractFileParams{FilePath: "archive.zip"}, device, "")
	if err != nil {
		t.Fatalf("ExtractFileImpl failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	got, err := os.ReadFile(filepath.Join(result.DestDir, "hello.txt"))
	if err != nil {
		t.Fatalf("expected hello.txt: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("hello.txt content = %q; want %q", got, "hello world")
	}

	got2, err := os.ReadFile(filepath.Join(result.DestDir, "subdir", "nested.txt"))
	if err != nil {
		t.Fatalf("expected subdir/nested.txt: %v", err)
	}
	if string(got2) != "nested content" {
		t.Errorf("nested.txt content = %q; want %q", got2, "nested content")
	}
}

func TestExtractFileImpl_PathTraversal(t *testing.T) {
	device := makeDevice(t)

	cases := []string{
		"../escape.txt",
		"../../escape.txt",
		"/absolute.txt",
		"subdir/../../escape.txt",
	}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			data := buildZip(t, []struct{ name, content string }{
				{name, "malicious"},
			})
			writeZipToFile(t, device.CirrusDir, "traversal.zip", data)

			_, err := ExtractFileImpl(ExtractFileParams{FilePath: "traversal.zip"}, device, "")
			if err == nil {
				t.Errorf("expected error for traversal entry %q, got nil", name)
			}
		})
	}
}

func TestExtractZipEntry_SizeLimit(t *testing.T) {
	const testLimit int64 = 10 // tiny limit for fast tests

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("big.txt")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	// Write testLimit+1 bytes to exceed the limit.
	if _, err := f.Write(bytes.Repeat([]byte("A"), int(testLimit+1))); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("open reader: %v", err)
	}

	destDir := t.TempDir()
	err = extractZipEntry(r.File[0], destDir, testLimit)
	if err == nil {
		t.Fatal("expected error for oversized entry, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractZip_EntryCountLimit(t *testing.T) {
	const testMaxEntries = 3

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for i := 0; i < testMaxEntries+1; i++ {
		f, err := w.Create(fmt.Sprintf("file_%d.txt", i))
		if err != nil {
			t.Fatalf("create entry %d: %v", i, err)
		}
		if _, err := f.Write([]byte("x")); err != nil {
			t.Fatalf("write entry %d: %v", i, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	rc, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("open reader: %v", err)
	}

	destDir := t.TempDir()
	_, err = extractZip(rc, destDir, testMaxEntries, MaxZipEntryBytes)
	if err == nil {
		t.Fatal("expected error for entry count limit, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds the limit") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractFileImpl_FileNotFound(t *testing.T) {
	device := makeDevice(t)

	_, err := ExtractFileImpl(ExtractFileParams{FilePath: "nonexistent.zip"}, device, "")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractFileImpl_NotAnArchive(t *testing.T) {
	device := makeDevice(t)

	if err := os.WriteFile(filepath.Join(device.CirrusDir, "doc.pdf"), []byte("%PDF"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ExtractFileImpl(ExtractFileParams{FilePath: "doc.pdf"}, device, "")
	if err == nil {
		t.Fatal("expected error for non-archive file")
	}
}

// --- tar.gz helpers ---

// buildTarGz creates an in-memory .tar.gz from (name, content) pairs.
func buildTarGz(t *testing.T, entries []struct{ name, content string }) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: tar.TypeReg,
			Size:     int64(len(e.content)),
			Mode:     0644,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("buildTarGz: write header %q: %v", e.name, err)
		}
		if _, err := tw.Write([]byte(e.content)); err != nil {
			t.Fatalf("buildTarGz: write content %q: %v", e.name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("buildTarGz: close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("buildTarGz: close gzip: %v", err)
	}
	return buf.Bytes()
}

func TestExtractFileImpl_TarGz_BasicExtraction(t *testing.T) {
	device := makeDevice(t)

	data := buildTarGz(t, []struct{ name, content string }{
		{"hello.txt", "hello from tar"},
		{"subdir/nested.txt", "nested tar content"},
	})
	if err := os.WriteFile(filepath.Join(device.CirrusDir, "archive.tar.gz"), data, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractFileImpl(ExtractFileParams{FilePath: "archive.tar.gz"}, device, "")
	if err != nil {
		t.Fatalf("ExtractFileImpl failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	got, err := os.ReadFile(filepath.Join(result.DestDir, "hello.txt"))
	if err != nil {
		t.Fatalf("expected hello.txt: %v", err)
	}
	if string(got) != "hello from tar" {
		t.Errorf("hello.txt = %q; want %q", got, "hello from tar")
	}

	got2, err := os.ReadFile(filepath.Join(result.DestDir, "subdir", "nested.txt"))
	if err != nil {
		t.Fatalf("expected subdir/nested.txt: %v", err)
	}
	if string(got2) != "nested tar content" {
		t.Errorf("nested.txt = %q; want %q", got2, "nested tar content")
	}
}

func TestExtractFileImpl_Tgz_BasicExtraction(t *testing.T) {
	device := makeDevice(t)

	data := buildTarGz(t, []struct{ name, content string }{
		{"file.txt", "tgz content"},
	})
	if err := os.WriteFile(filepath.Join(device.CirrusDir, "archive.tgz"), data, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractFileImpl(ExtractFileParams{FilePath: "archive.tgz"}, device, "")
	if err != nil {
		t.Fatalf("ExtractFileImpl (.tgz) failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(result.DestDir, "file.txt"))
	if err != nil {
		t.Fatalf("expected file.txt: %v", err)
	}
	if string(got) != "tgz content" {
		t.Errorf("file.txt = %q; want %q", got, "tgz content")
	}
}

func TestExtractFileImpl_TarGz_PathTraversal(t *testing.T) {
	device := makeDevice(t)

	cases := []string{
		"../escape.txt",
		"/absolute.txt",
	}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			data := buildTarGz(t, []struct{ name, content string }{
				{name, "malicious"},
			})
			archiveName := "traversal.tar.gz"
			if err := os.WriteFile(filepath.Join(device.CirrusDir, archiveName), data, 0644); err != nil {
				t.Fatal(err)
			}

			_, err := ExtractFileImpl(ExtractFileParams{FilePath: archiveName}, device, "")
			if err == nil {
				t.Errorf("expected traversal error for %q, got nil", name)
			}
		})
	}
}

func TestExtractTarEntry_SizeLimit(t *testing.T) {
	const testLimit int64 = 10

	content := bytes.Repeat([]byte("A"), int(testLimit+1))
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name:     "big.txt",
		Typeflag: tar.TypeReg,
		Size:     int64(len(content)),
		Mode:     0644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	tr := tar.NewReader(gr)
	nextHdr, err := tr.Next()
	if err != nil {
		t.Fatalf("tar next: %v", err)
	}

	destDir := t.TempDir()
	err = extractTarEntry(nextHdr, tr, destDir, testLimit)
	if err == nil {
		t.Fatal("expected size limit error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractTar_EntryCountLimit(t *testing.T) {
	const testMax = 3

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for i := 0; i < testMax+1; i++ {
		content := []byte("x")
		hdr := &tar.Header{
			Name:     fmt.Sprintf("file_%d.txt", i),
			Typeflag: tar.TypeReg,
			Size:     int64(len(content)),
			Mode:     0644,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %d: %v", i, err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("write content %d: %v", i, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	destDir := t.TempDir()
	_, err = extractTar(tar.NewReader(gr), destDir, testMax, MaxZipEntryBytes)
	if err == nil {
		t.Fatal("expected entry count limit error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestArchiveExt(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"archive.zip", ".zip"},
		{"archive.tar.gz", ".tar.gz"},
		{"ARCHIVE.TAR.GZ", ".tar.gz"},
		{"archive.tgz", ".tgz"},
		{"archive.tar.bz2", ".tar.bz2"},
		{"archive.rar", ".rar"},
		{"file.txt", ".txt"},
	}
	for _, c := range cases {
		got := archiveExt(c.path)
		if got != c.want {
			t.Errorf("archiveExt(%q) = %q; want %q", c.path, got, c.want)
		}
	}
}

func TestArchiveDestDir_DoubleExtension(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		filename string
		wantStem string
	}{
		{"foo.tar.gz", "foo"},
		{"foo.tgz", "foo"},
		{"foo.zip", "foo"},
		{"foo.tar.bz2", "foo"},
	}
	for _, c := range cases {
		fullPath := filepath.Join(dir, c.filename)
		got := archiveDestDir(fullPath)
		gotBase := filepath.Base(got)
		// May have a suffix like " (1)" if the path exists — just check prefix
		if gotBase != c.wantStem && !strings.HasPrefix(gotBase, c.wantStem+" ") {
			t.Errorf("archiveDestDir(%q) base = %q; want stem %q", c.filename, gotBase, c.wantStem)
		}
	}
}
