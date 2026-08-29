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
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func init() {
	var err error
	client, err = azblob.NewClientWithNoCredential(DefaultUpdateSources[0].BaseUrl(), nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Azure Blob client: %v", err))
	}
}

// fetchURL performs a GET and returns the response body as bytes.
// Only HTTPS connections to known update hosts are allowed.
// Returns errNotFound on HTTP 404; other non-200 responses return a
// descriptive error.
func fetchURL(rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" && !allowHTTPInFetchURL {
		return nil, fmt.Errorf("insecure URL scheme %q: only https is allowed for downloads", parsed.Scheme)
	}

	// Validate host and replace it with the canonical allowlist value so the
	// request URL is constructed from a compile-time constant, not user input.
	// This breaks the taint flow that CodeQL's SSRF rule tracks.
	canonicalHost, ok := canonicalUpdateHost(parsed.Hostname())
	if !ok && !allowHTTPInFetchURL {
		return nil, fmt.Errorf("host %q is not an allowed update server", parsed.Hostname())
	}

	// Build the request URL from the allowlist-sourced host (untainted) plus
	// the path/query from the parsed input.
	scheme := "https"
	if allowHTTPInFetchURL && parsed.Scheme == "http" {
		scheme = "http"
	}
	var requestHost string
	if ok {
		// Use the canonical constant from the allowlist — not the user-supplied value.
		requestHost = canonicalHost
		if parsed.Port() != "" {
			requestHost = canonicalHost + ":" + parsed.Port()
		}
	} else {
		// allowHTTPInFetchURL path (tests only): use the raw host as-is.
		requestHost = parsed.Host
	}
	safeURL := &url.URL{
		Scheme:   scheme,
		Host:     requestHost,
		Path:     parsed.EscapedPath(),
		RawQuery: parsed.RawQuery,
	}
	req, err := http.NewRequest(http.MethodGet, safeURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// URL is safe: scheme enforced to https, host validated against an explicit
	// allowlist and replaced with the matching compile-time constant, path and
	// query are URL-encoded release asset components. go/ssrf is excluded in
	// .github/codeql/codeql-go-config.yml.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, requestHost)
	}
	return io.ReadAll(resp.Body)
}

// canonicalUpdateHost returns the allowlist entry that matches host, and true
// if found. The returned string is a compile-time constant from allowedUpdateHosts,
// not derived from user input — which breaks the taint chain for SSRF analysis.
func canonicalUpdateHost(host string) (string, bool) {
	for _, allowed := range allowedUpdateHosts {
		if host == allowed {
			return allowed, true
		}
	}
	return "", false
}

// isAllowedUpdateHost reports whether host is in the update server allowlist.
func isAllowedUpdateHost(host string) bool {
	_, ok := canonicalUpdateHost(host)
	return ok
}

// verifyChecksum fetches a .sha256 file from checksumURL and compares it
// against the SHA-256 hash of data. The .sha256 file may contain the bare
// hex digest or a line in the format produced by sha256sum(1):
//
//	<hex>  <filename>
func verifyChecksum(data []byte, checksumURL string) error {
	checksumBytes, err := fetchURL(checksumURL)
	if err != nil {
		if errors.Is(err, errNotFound) {
			// Only *here* does a 404 mean "no checksum was published".
			return errChecksumUnavailable
		}
		return err
	}

	line := strings.TrimSpace(string(checksumBytes))
	// sha256sum(1) format: "<hex>  <filename>" — take only the hex part.
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return errors.New("checksum file is empty")
	}
	expected, err := hex.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid checksum format: %w", err)
	}

	actual := sha256.Sum256(data)
	if !hmacEqual(actual[:], expected) {
		return fmt.Errorf("checksum mismatch: expected %x, got %x", expected, actual)
	}
	return nil
}

// hmacEqual is a constant-time comparison of two byte slices.
func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func backupSelf() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	tmpFile, err := os.CreateTemp("", backupName)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	src, err := os.Open(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to open current binary: %w", err)
	}
	defer src.Close()

	if _, err := src.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek in current binary: %w", err)
	}
	if _, err := tmpFile.ReadFrom(src); err != nil {
		return "", fmt.Errorf("failed to copy binary to temp: %w", err)
	}
	return tmpFile.Name(), nil
}

func replaceSelf(body io.Reader) error {
	execPath, err := resolvedExecutable()
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp("", "quark_update_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for update: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.ReadFrom(body); err != nil {
		return fmt.Errorf("failed to write update to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	// Rewind the temp file to the beginning
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek in temp file: %w", err)
	}

	gzReader, err := gzip.NewReader(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var binFile *os.File
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}
		if header.Typeflag == tar.TypeReg && (header.Name == binaryName || header.Name == fmt.Sprintf("./%s", binaryName)) {
			binFile, err = os.CreateTemp("", extractedName)
			if err != nil {
				return fmt.Errorf("failed to create temp file for extracted binary: %w", err)
			}
			if _, err := io.Copy(binFile, tarReader); err != nil {
				return fmt.Errorf("failed to extract binary from tar: %w", err)
			}
			if err := binFile.Sync(); err != nil {
				return fmt.Errorf("failed to sync extracted binary: %w", err)
			}
			break
		}
	}
	if binFile == nil {
		return errors.New("binary not found in archive")
	}
	defer os.Remove(binFile.Name())
	defer binFile.Close()

	// Make the extracted binary executable
	if err := os.Chmod(binFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Close the file before operations
	binFile.Close()

	// Create a temporary file in the same directory as the target executable.
	// This keeps the subsequent rename atomic and on the same filesystem, so it
	// must not be relocated to /tmp — that would trade a permission failure for
	// a torn-binary failure. CanSelfUpdate checks this is writable up front.
	execDir := filepath.Dir(execPath)
	tmpNew, err := os.CreateTemp(execDir, ".quark_new_*")
	if err != nil {
		return fmt.Errorf(
			"%w: failed to create temp file in %s: %w",
			ErrSelfUpdateUnavailable, execDir, err,
		)
	}
	tmpNewPath := tmpNew.Name()
	defer os.Remove(tmpNewPath)

	// Copy the new binary to the target filesystem
	src, err := os.Open(binFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open extracted binary: %w", err)
	}
	defer src.Close()

	if _, err := io.Copy(tmpNew, src); err != nil {
		tmpNew.Close()
		return fmt.Errorf("failed to copy new binary: %w", err)
	}

	if err := tmpNew.Sync(); err != nil {
		tmpNew.Close()
		return fmt.Errorf("failed to sync new binary: %w", err)
	}
	tmpNew.Close()

	// Set executable permissions on the new file
	if err := os.Chmod(tmpNewPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions on new binary: %w", err)
	}

	// Atomically rename the new binary over the old one
	// This works even while the old binary is running
	if err := os.Rename(tmpNewPath, execPath); err != nil {
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	return nil
}

// selfUpdateDir returns the directory holding the running executable — the
// directory replaceSelf must be able to write to.
//
// Symlinks are resolved so that the answer is the directory the atomic rename
// actually lands in. With the binary installed at serviceBinaryPath and
// /usr/local/bin/quark kept as a symlink, the resolved directory is the
// group-writable one, not the root-owned one the symlink lives in.
func selfUpdateDir() (string, error) {
	execPath, err := resolvedExecutable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(execPath), nil
}

// resolvedExecutable returns the path of the running binary with symlinks
// resolved, so the update lands on the real file rather than replacing a
// symlink with a regular file.
//
// On Linux os.Executable already resolves (/proc/self/exe); on darwin it can
// hand back the path as invoked. With /usr/local/bin/quark kept as a symlink
// into the self-updatable directory, that difference decides whether an update
// replaces the binary or clobbers the symlink.
func resolvedExecutable() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		// Keep the unresolved path: it is still the best available answer, and
		// the write probe decides whether it is usable.
		return execPath, nil
	}
	return resolved, nil
}

// canSelfUpdateIn is CanSelfUpdate against an explicit directory, so the
// failure path can be exercised without a binary installed in a directory it cannot write.
func canSelfUpdateIn(dir string) error {
	probe, err := os.CreateTemp(dir, ".quark_preflight_*")
	if err != nil {
		return fmt.Errorf(
			"%w: %s is not writable by %s.\n"+
				"Replacing a running binary in place requires write access to the directory "+
				"holding it.\n"+
				"Fix it by running `sudo quark install`, which moves the binary to %s and "+
				"leaves %s as a symlink to it, or update through your package manager or OS "+
				"image instead.\n"+
				"Underlying error: %w",
			ErrSelfUpdateUnavailable, dir, currentAccountDescription(),
			SelfUpdatableBinDir, LegacyBinPath, err,
		)
	}
	name := probe.Name()
	probe.Close()
	if err := os.Remove(name); err != nil {
		fmt.Printf("Warning: failed to remove preflight probe file %s: %v\n", name, err)
	}
	return nil
}

// currentAccountDescription names the account this process runs as, for error
// messages. os/user can fail in a static build with no cgo, so the numeric uid
// is the fallback — it is what `systemctl show quark -p User` will be compared
// against either way.
func currentAccountDescription() string {
	uid := os.Geteuid()
	if u, err := user.Current(); err == nil && u.Username != "" {
		return fmt.Sprintf("user %s (uid %d)", u.Username, uid)
	}
	return fmt.Sprintf("uid %d", uid)
}
