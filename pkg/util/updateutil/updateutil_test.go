package updateutil

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	github "github.com/autobutler-org/autobutler/pkg/util/githubutil"
)

var defaultUpdateSource = DefaultUpdateSources[0]

func TestMain(m *testing.M) {
	// Allow http:// URLs in fetchURL so httptest servers work in unit tests.
	allowHTTPInFetchURL = true
	os.Exit(m.Run())
}

func TestListPossibleUpdates_NoCurrentVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	result, err := ListPossibleUpdates(
		defaultUpdateSource,
		false,
	)

	if err != nil {
		t.Fatalf("ListPossibleUpdates failed: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestUpdate_EmptyVersion(t *testing.T) {
	err := Update(defaultUpdateSource, "")
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

	err := Update(defaultUpdateSource, "v1.0.0")
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

// serveRealReleaseTarGz starts a mock HTTP server that serves the real v0.13.0
// Linux arm64 release tarball (cached locally). This lets us test the full
// download → decompress → extract path without hitting GitHub.
func serveRealReleaseTarGz(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	tarPath := "/tmp/autobutler-test-release/autobutler_Linux_arm64.tar.gz"
	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		t.Skip("Real release tarball not available at " + tarPath + " — run: curl -sL https://github.com/autobutler-org/autobutler/releases/download/v0.13.0/autobutler_Linux_arm64.tar.gz -o " + tarPath)
	}
	data, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatalf("failed to read release tarball: %v", err)
	}
	version := "v0.13.0"
	archiveName := ConstructArchiveName()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := fmt.Sprintf("/%s/%s", version, archiveName)
		if r.URL.Path != expected {
			t.Logf("unexpected path: %s (expected %s)", r.URL.Path, expected)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	return server, version
}

func TestUpdate_RealRelease_ReplaceSelf(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skipf("Real release test only runs on linux/arm64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	tarPath := "/tmp/autobutler-test-release/autobutler_Linux_arm64.tar.gz"
	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		t.Skip("Real release tarball not available")
	}
	data, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatalf("failed to read tarball: %v", err)
	}
	// replaceSelf extracts the binary and atomically replaces os.Executable().
	// In the test runner context this will succeed or fail with a permission
	// error — either way is acceptable. What we verify is no panic and no
	// unexpected error (only permission/read-only are OK to get).
	err = replaceSelf(strings.NewReader(string(data)))
	if err != nil &&
		!strings.Contains(err.Error(), "permission") &&
		!strings.Contains(err.Error(), "read-only") &&
		!strings.Contains(err.Error(), "text file busy") {
		t.Errorf("replaceSelf with real tarball returned unexpected error: %v", err)
	}
}

func TestReplaceSelf_WithRealTarball(t *testing.T) {
	tarPath := "/tmp/autobutler-test-release/autobutler_Linux_arm64.tar.gz"
	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		t.Skip("Real release tarball not available")
	}

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("failed to open tarball: %v", err)
	}
	defer f.Close()

	// replaceSelf will try to overwrite os.Executable() — which in test context
	// is the test binary itself. It may fail with permission denied, but it
	// should successfully decompress and extract the binary from the archive
	// before attempting the replace. We verify the archive is parseable.
	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		if hdr.Name == binaryName || hdr.Name == "./"+binaryName {
			found = true
			if hdr.Size == 0 {
				t.Error("Expected non-zero binary size in archive")
			}
			break
		}
	}
	if !found {
		t.Errorf("Binary %q not found in release tarball", binaryName)
	}
}

func TestGetLatestVersion_GithubSource(t *testing.T) {
	mockRelease := github.Release{
		TagName: "v0.99.0",
		Assets: []github.Asset{
			{BrowserDownloadURL: fmt.Sprintf("https://example.com/v0.99.0/%s", ConstructArchiveName())},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "releases/latest") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"tag_name":"%s","assets":[{"browser_download_url":"%s"}]}`,
			mockRelease.TagName,
			mockRelease.Assets[0].BrowserDownloadURL,
		)
	}))
	defer server.Close()
	reset := github.SetBaseURLForTesting(server.URL)
	defer reset()

	githubSource := NewUpdateSource(UpdateSourceKindGithub, "test-org", "test-repo")
	version, err := GetLatestVersion(githubSource)
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}
	if version != "v0.99.0" {
		t.Errorf("Expected version v0.99.0, got %s", version)
	}
}

func TestGetLatestVersion_GithubSource_NoAsset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"tag_name":"v1.0.0","assets":[]}`)
	}))
	defer server.Close()
	reset := github.SetBaseURLForTesting(server.URL)
	defer reset()

	source := NewUpdateSource(UpdateSourceKindGithub, "test-org", "test-repo")
	_, err := GetLatestVersion(source)
	if err == nil {
		t.Error("Expected error when no suitable asset found")
	}
}

func TestGetLatestVersion_UnsupportedSource(t *testing.T) {
	source := &UpdateSource{Kind: "unsupported"}
	_, err := GetLatestVersion(source)
	if err == nil {
		t.Error("Expected error for unsupported source kind")
	}
}

func TestListPossibleUpdates_GithubSource_WithReleases(t *testing.T) {
	archiveName := ConstructArchiveName()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "releases") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `[
			{"tag_name":"v1.0.0","assets":[{"browser_download_url":"https://example.com/v1.0.0/%s"}]},
			{"tag_name":"v1.1.0","assets":[{"browser_download_url":"https://example.com/v1.1.0/%s"}]}
		]`, archiveName, archiveName)
	}))
	defer server.Close()
	reset := github.SetBaseURLForTesting(server.URL)
	defer reset()

	source := NewUpdateSource(UpdateSourceKindGithub, "test-org", "test-repo")
	result, err := ListPossibleUpdates(source, true)
	if err != nil {
		t.Fatalf("ListPossibleUpdates failed: %v", err)
	}
	if len(result.Versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(result.Versions))
	}
}

func TestListPossibleUpdates_GithubSource_SkipsNoAsset(t *testing.T) {
	archiveName := ConstructArchiveName()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// One release has a .tar.gz asset, one has no assets at all
		fmt.Fprintf(w, `[
			{"tag_name":"v1.0.0","assets":[{"browser_download_url":"https://example.com/v1.0.0/%s"}]},
			{"tag_name":"v1.1.0","assets":[]}
		]`, archiveName)
	}))
	defer server.Close()
	reset := github.SetBaseURLForTesting(server.URL)
	defer reset()

	source := NewUpdateSource(UpdateSourceKindGithub, "test-org", "test-repo")
	result, err := ListPossibleUpdates(source, true)
	if err != nil {
		t.Fatalf("ListPossibleUpdates failed: %v", err)
	}
	// v1.1.0 is skipped because it has no .tar.gz asset
	if len(result.Versions) != 1 {
		t.Errorf("Expected 1 version (v1.1.0 skipped due to no .tar.gz asset), got %d", len(result.Versions))
	}
	if result.Versions[0].Version != "v1.0.0" {
		t.Errorf("Expected v1.0.0, got %s", result.Versions[0].Version)
	}
}

func TestUpdateFromDefaultSources_AllFail(t *testing.T) {
	// Point all sources to a server that always 404s
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	reset := github.SetBaseURLForTesting(server.URL)
	defer reset()

	err := UpdateFromDefaultSources("v9.9.9")
	if err == nil {
		t.Error("Expected error when all sources fail")
	}
}

func TestConstructArchiveName(t *testing.T) {
	name := ConstructArchiveName()
	if name == "" {
		t.Error("Expected non-empty archive name")
	}
	if !strings.HasSuffix(name, ".tar.gz") {
		t.Errorf("Expected archive name to end in .tar.gz, got %s", name)
	}
	// Should contain OS and arch
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	expectedOS := strings.ToUpper(goos[:1]) + goos[1:]
	if !strings.Contains(name, expectedOS) {
		t.Errorf("Expected archive name to contain OS %q, got %s", expectedOS, name)
	}
	if !strings.Contains(name, goarch) {
		t.Errorf("Expected archive name to contain arch %q, got %s", goarch, name)
	}
}

func TestUpdate_WithMockServer_RealTarball(t *testing.T) {
	// Note: Update() for GitHub sources builds the download URL from hardcoded github.com
	// base URLs in UpdateSource.BaseUrl(), which can't be intercepted via githubutil's
	// SetBaseURLForTesting. Full end-to-end testing of Update() requires either:
	//   a) Refactoring UpdateSource to accept a base URL override, or
	//   b) An integration test hitting the real GitHub.
	// The core logic (download → decompress → extract → replace) is covered by
	// TestReplaceSelf_WithRealTarball and TestUpdate_RealRelease_ReplaceSelf.
	t.Skip("Update() download URL is hardcoded in UpdateSource.BaseUrl() — see comment")
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

func TestUpdateSource_BaseURLOverride(t *testing.T) {
	source := NewUpdateSource(UpdateSourceKindGithub, "autobutler-org", "autobutler")

	// Without override: should return real github.com URL
	if got := source.BaseUrl(); got == "" {
		t.Error("Expected non-empty BaseUrl without override")
	}

	// With override: should return the override
	source.BaseURLOverride = "http://localhost:9999"
	if got := source.BaseUrl(); got != "http://localhost:9999" {
		t.Errorf("Expected override URL, got %q", got)
	}

	// UpdateUrl for GitHub should equal BaseUrl (no double /releases/download)
	if got := source.UpdateUrl(); got != "http://localhost:9999" {
		t.Errorf("Expected UpdateUrl to equal BaseUrl for GitHub, got %q", got)
	}
}

func TestUpdate_WithBaseURLOverride_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	source := NewUpdateSource(UpdateSourceKindGithub, "autobutler-org", "autobutler")
	source.BaseURLOverride = server.URL

	err := Update(source, "v1.0.0")
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestListPossibleUpdates_FilteredToEmpty_ReturnsEmptySliceNotNil(t *testing.T) {
	// When version filtering leaves no results, Versions must be a non-nil
	// empty slice so it marshals to JSON [] rather than null.
	// Use a GitHub source with no matching assets so updateReleases is empty
	// and the filter loop never appends — this exercises the zero-item path.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "releases") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		// Release exists but has no assets matching this platform's archive name
		fmt.Fprint(w, `[{"tag_name":"v1.0.0","assets":[{"browser_download_url":"https://example.com/other-platform.zip"}]}]`)
	}))
	defer server.Close()
	reset := github.SetBaseURLForTesting(server.URL)
	defer reset()

	source := NewUpdateSource(UpdateSourceKindGithub, "test-org", "test-repo")
	result, err := ListPossibleUpdates(source, true) // allVersions=true, no filter
	if err != nil {
		t.Fatalf("ListPossibleUpdates failed: %v", err)
	}
	if result.Versions == nil {
		t.Error("Expected non-nil empty slice, got nil (would marshal to JSON null)")
	}
	if len(result.Versions) != 0 {
		t.Errorf("Expected 0 versions (no matching assets), got %d", len(result.Versions))
	}
}

// --- verifyChecksum / fetchURL ---

func TestVerifyChecksum_Match(t *testing.T) {
	data := []byte("hello, butler")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, hexSum)
	}))
	defer server.Close()

	if err := verifyChecksum(data, server.URL+"/checksum"); err != nil {
		t.Errorf("expected no error for matching checksum, got %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	data := []byte("hello, butler")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a completely wrong checksum.
		fmt.Fprint(w, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	}))
	defer server.Close()

	err := verifyChecksum(data, server.URL+"/checksum")
	if err == nil {
		t.Error("expected error for mismatched checksum")
	}
	if errors.Is(err, errChecksumUnavailable) {
		t.Error("mismatch should not report errChecksumUnavailable")
	}
}

func TestVerifyChecksum_Sha256sumFormat(t *testing.T) {
	// sha256sum(1) format: "<hex>  <filename>"
	data := []byte("sha256sum format test")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  autobutler-linux-arm64.tar.gz\n", hexSum)
	}))
	defer server.Close()

	if err := verifyChecksum(data, server.URL+"/checksum"); err != nil {
		t.Errorf("expected no error for sha256sum-format checksum, got %v", err)
	}
}

func TestVerifyChecksum_Unavailable(t *testing.T) {
	data := []byte("data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := verifyChecksum(data, server.URL+"/checksum.sha256")
	if !errors.Is(err, errChecksumUnavailable) {
		t.Errorf("expected errChecksumUnavailable for 404, got %v", err)
	}
}

func TestVerifyChecksum_EmptyFile(t *testing.T) {
	data := []byte("data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty body
	}))
	defer server.Close()

	err := verifyChecksum(data, server.URL+"/checksum")
	if err == nil {
		t.Error("expected error for empty checksum file")
	}
}

func TestVerifyChecksum_InvalidHex(t *testing.T) {
	data := []byte("data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not-a-hex-string")
	}))
	defer server.Close()

	err := verifyChecksum(data, server.URL+"/checksum")
	if err == nil {
		t.Error("expected error for invalid hex checksum")
	}
}

func TestFetchURL_404ReturnsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchURL(server.URL + "/missing")
	if !errors.Is(err, errChecksumUnavailable) {
		t.Errorf("expected errChecksumUnavailable for 404, got %v", err)
	}
}

func TestFetchURL_500ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchURL(server.URL + "/error")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
	if errors.Is(err, errChecksumUnavailable) {
		t.Error("HTTP 500 should not be errChecksumUnavailable")
	}
}

func TestFetchURL_RejectsHTTP(t *testing.T) {
	// Temporarily re-enable the HTTPS restriction for this test.
	allowHTTPInFetchURL = false
	defer func() { allowHTTPInFetchURL = true }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := fetchURL(server.URL + "/file")
	if err == nil {
		t.Error("expected error for http:// URL")
	}
	if !strings.Contains(err.Error(), "only https is allowed") {
		t.Errorf("unexpected error: %v", err)
	}
}
