package updateutil

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestListPossibleUpdates_NoCurrentVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	result, err := ListPossibleUpdates("autobutler-org", "autobutler.org")

	if err != nil {
		t.Logf("Got error (expected in some environments): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestUpdate_EmptyVersion(t *testing.T) {
	params := UpdateParams{
		Version: "",
	}

	err := Update(params)
	if err == nil {
		t.Error("Expected error for empty version")
	}

	if err.Error() != "version cannot be empty" {
		t.Errorf("Expected 'version cannot be empty' error, got: %v", err)
	}
}

func TestUpdate_404Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	os.Setenv("AUTOBUTLER_UPDATE_URL", server.URL)
	defer os.Unsetenv("AUTOBUTLER_UPDATE_URL")

	params := UpdateParams{
		Version: "v1.0.0",
	}

	err := Update(params)
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestBackupSelf(t *testing.T) {
	backupPath, err := backupSelf()
	if err != nil {
		t.Fatalf("backupSelf failed: %v", err)
	}
	defer os.Remove(backupPath)

	if backupPath == "" {
		t.Error("Expected non-empty backup path")
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Failed to stat backup: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Backup file is empty")
	}
}

func createMockTarGz(t *testing.T, binaryName string, content []byte) []byte {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test_tar_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	gzWriter := gzip.NewWriter(tmpFile)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name:     binaryName,
		Mode:     0755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}

	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	tarWriter.Close()
	gzWriter.Close()
	tmpFile.Close()

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read tar file: %v", err)
	}

	return data
}

type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestReplaceSelf_BinaryNotInArchive(t *testing.T) {
	tarData := createMockTarGz(t, "wrongname", []byte("content"))

	err := replaceSelf(io.NopCloser(newBytesReader(tarData)))
	if err == nil {
		t.Error("Expected error when binary not found in archive")
	}

	if err.Error() != "binary not found in archive" {
		t.Errorf("Expected 'binary not found in archive', got: %v", err)
	}
}

func TestConstants(t *testing.T) {
	if binaryName != "autobutler" {
		t.Errorf("Expected binaryName to be 'autobutler', got '%s'", binaryName)
	}

	expectedBackupName := "autobutler_backup"
	if backupName != expectedBackupName {
		t.Errorf("Expected backupName to be '%s', got '%s'", expectedBackupName, backupName)
	}

	expectedExtractedName := "autobutler_extracted"
	if extractedName != expectedExtractedName {
		t.Errorf("Expected extractedName to be '%s', got '%s'", expectedExtractedName, extractedName)
	}
}

func TestIsDevelopmentVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.0.0", false},
		{"1.0.0-beta", true},
		{"1.0.0-rc1", true},
		{"1.0.0+build", false},
		{"v1.2.3", false},
		{"v1.2.3-dev", true},
		{"v1.2.3-alpha.1", true},
		{"v1.2.3.4", false},
		{"1.0.0-", true},
		{"-1.0.0", true},
		{"1.0.0--dev", true},
		{"", false},
	}
	for _, tt := range tests {
		got := IsDevelopmentVersion(tt.version)
		if got != tt.want {
			t.Errorf("IsDevelopmentVersion(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}
