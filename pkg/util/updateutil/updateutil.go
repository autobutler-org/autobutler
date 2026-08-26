package updateutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
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
	"regexp"
	"runtime"
	"slices"
	"strings"

	github "github.com/autobutler-org/quark/pkg/util/githubutil"
	"github.com/autobutler-org/quark/pkg/util/versionutil"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var client *azblob.Client

func init() {
	var err error
	client, err = azblob.NewClientWithNoCredential(DefaultUpdateSources[0].BaseUrl(), nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Azure Blob client: %v", err))
	}
}

func ConstructArchiveName() string {
	goos := fmt.Sprintf("%s%s", strings.ToUpper(string(runtime.GOOS[0])), string(runtime.GOOS[1:]))
	return fmt.Sprintf("quark_%s_%s.tar.gz", goos, runtime.GOARCH)
}

func GetLatestVersionFromDefaultSources() (string, error) {
	for _, source := range DefaultUpdateSources {
		version, err := GetLatestVersion(source)
		if err != nil {
			continue
		}
		return version, nil
	}
	return "", errors.New("failed to get latest version from all default sources")
}

func GetLatestVersion(source *UpdateSource) (string, error) {
	if source == nil {
		return GetLatestVersionFromDefaultSources()
	}
	switch source.Kind {
	case UpdateSourceKindGithub:
		org, repo := source.Account, source.Path
		release, err := github.FetchLatestRelease(org, repo)
		if err != nil {
			return "", fmt.Errorf("failed to fetch releases: %w", err)
		}

		url := getAssetURLFromRelease(release)
		if url == "" {
			return "", errors.New("no suitable asset found in latest release")
		}

		return release.TagName, nil
	case UpdateSourceKindAzure:
		releases, err := ListPossibleUpdates(source, true)
		if err != nil {
			return "", fmt.Errorf("failed to list releases: %w", err)
		}
		if len(releases.Versions) == 0 {
			return "", errors.New("no releases found in Azure source")
		}
		// Sort by semver and return the highest. Lexicographic order is not
		// reliable for semver (e.g. "v0.9.0" > "v0.16.0" lexicographically).
		latest := releases.Versions[0]
		for _, v := range releases.Versions[1:] {
			if versionutil.CompareVersions(
				versionutil.Version{Semver: v.Version},
				versionutil.Version{Semver: latest.Version},
			) == 1 {
				latest = v
			}
		}
		return latest.Version, nil
	default:
		return "", fmt.Errorf("unsupported update source kind: %s", source.Kind)
	}
}

func IsDevelopmentVersion(version string) bool {
	if matched, err := regexp.MatchString("^.*-", version); err != nil || matched {
		return true
	}
	if matched, err := regexp.MatchString("-.*$", version); err != nil || matched {
		return true
	}
	return false
}

// ListPossibleUpdatesResult contains the result of listing possible updates
type ListPossibleUpdatesResult struct {
	Versions []*UpdateVersion
}

// ListPossibleUpdatesFromDefaultSources lists possible updates from all default sources
func ListPossibleUpdatesFromDefaultSources(allVersions bool) (*ListPossibleUpdatesResult, error) {
	for _, source := range DefaultUpdateSources {
		result, err := ListPossibleUpdates(source, allVersions)
		if err != nil {
			continue
		}
		return result, nil
	}
	return nil, errors.New("failed to list updates from all default sources")
}

// ListPossibleUpdates retrieves all available releases that are newer than the current version
func ListPossibleUpdates(source *UpdateSource, allVersions bool) (*ListPossibleUpdatesResult, error) {
	if source == nil {
		return ListPossibleUpdatesFromDefaultSources(allVersions)
	}

	updateReleases := []*UpdateVersion{}
	switch source.Kind {
	case UpdateSourceKindGithub:
		org, repo := source.Account, source.Path
		releases, err := github.FetchReleases(org, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch releases: %w", err)
		}
		for _, release := range releases {
			url := getAssetURLFromRelease(release)
			if url == "" {
				continue
			}
			updateReleases = append(updateReleases, &UpdateVersion{
				Version: release.TagName,
				URL:     url,
			})
		}
	case UpdateSourceKindAzure:
		prefix := source.BlobPrefix()
		pager := client.NewListBlobsFlatPager(
			source.Container(),
			&azblob.ListBlobsFlatOptions{
				Prefix: prefix,
			},
		)

		for pager.More() {
			resp, err := pager.NextPage(context.Background())
			if err != nil {
				return nil, fmt.Errorf("failed to list blobs: %w", err)
			}

			for _, blob := range resp.Segment.BlobItems {
				trimmed := strings.TrimPrefix(*blob.Name, *prefix)
				split := strings.Split(trimmed, "/")
				version, artifact := split[0], split[1]
				if strings.HasSuffix(artifact, ".txt") {
					continue
				}
				if !strings.Contains(artifact, ConstructArchiveName()) {
					continue
				}
				updateReleases = append(updateReleases, &UpdateVersion{
					Version: version,
					URL:     fmt.Sprintf("%s/%s/%s", source.BaseUrl(), source.Container(), *blob.Name),
				})
			}
		}
		slices.Reverse(updateReleases)
	default:
		return nil, fmt.Errorf("unsupported update source kind: %s", source.Kind)
	}

	if allVersions {
		return &ListPossibleUpdatesResult{
			Versions: updateReleases,
		}, nil
	}

	currentVersion := versionutil.GetVersion()
	if currentVersion.Semver == "" {
		return &ListPossibleUpdatesResult{
			Versions: updateReleases,
		}, nil
	}

	possibleUpdates := make([]*UpdateVersion, 0)
	for _, release := range updateReleases {
		if IsDevelopmentVersion(release.Version) {
			continue
		}
		comparison := versionutil.CompareVersions(
			versionutil.Version{
				Semver: release.Version,
			},
			*currentVersion,
		)
		if comparison > 0 {
			possibleUpdates = append(possibleUpdates, release)
		}
	}

	return &ListPossibleUpdatesResult{
		Versions: possibleUpdates,
	}, nil
}

// UpdateFromDefaultSources tries to update from all default sources until one succeeds
func UpdateFromDefaultSources(version string) error {
	// Checked once, before the loop. Without this the updater downloaded and
	// extracted the whole binary once per source before discovering it could
	// not write the result anywhere (#1609).
	if err := CanSelfUpdate(); err != nil {
		return err
	}

	errs := []error{}
	for _, source := range DefaultUpdateSources {
		fmt.Printf("Attempting to update from source: %v, with URL %s\n", source, source.UpdateUrl())
		err := Update(source, version)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update from source %v: %w", source, err))
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to update from all default sources: %v", errs)
}

// Update downloads and installs a new version of the application.
// If a companion .sha256 file is available at the same URL prefix, the
// archive is verified against it before replacing the running binary.
// A missing checksum file is treated as a warning (logged) rather than a
// hard failure, to keep compatibility with update sources that don't yet
// publish checksums.
func Update(source *UpdateSource, version string) error {
	if source == nil {
		return UpdateFromDefaultSources(version)
	}

	if version == "" {
		return errors.New("version cannot be empty")
	}

	// Before the backup copy and the download, not after: both are wasted work
	// if the binary cannot be replaced at the end of it (#1609).
	if err := CanSelfUpdate(); err != nil {
		return err
	}

	_, err := backupSelf()
	if err != nil {
		return fmt.Errorf("failed to copy current binary: %w", err)
	}

	baseUrl := source.UpdateUrl()
	if baseUrl == "" {
		return fmt.Errorf("invalid update source: %s", source.Kind)
	}

	archiveName := ConstructArchiveName()
	url := fmt.Sprintf("%s/%s/%s", baseUrl, version, archiveName)
	fmt.Println("Downloading update from", url)

	// Download the archive into memory so we can verify its checksum before
	// overwriting the running binary.
	archiveBytes, err := fetchURL(url)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return fmt.Errorf(
				"no release asset %s for version %s at %s: the version, the release, "+
					"or the update source itself does not exist: %w",
				archiveName, version, source, err,
			)
		}
		return fmt.Errorf("failed to download update from %s: %w", url, err)
	}

	// Attempt checksum verification. A 404 on the .sha256 file is treated as
	// "checksum unavailable" (warning only) to stay compatible with older
	// release assets that predate this feature.
	checksumURL := url + ".sha256"
	if err := verifyChecksum(archiveBytes, checksumURL); err != nil {
		if errors.Is(err, errChecksumUnavailable) {
			fmt.Println("Warning: no checksum file available at", checksumURL, "— skipping verification")
		} else {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	if err := replaceSelf(bytes.NewReader(archiveBytes)); err != nil {
		return fmt.Errorf("failed to replace self with update from %s: %w", url, err)
	}

	fmt.Println("Update successful.")
	return nil
}

// errNotFound is returned by fetchURL on HTTP 404. It carries no opinion about
// what was missing — callers wrap it with the meaning for their own request.
// It used to be spelled errChecksumUnavailable and returned for *every* 404,
// so a download from a repository that does not exist reported "checksum file
// not found" and sent readers looking at release assets (#1610).
var errNotFound = errors.New("not found (HTTP 404)")

// errChecksumUnavailable means the archive downloaded fine but no .sha256
// companion file was published alongside it.
var errChecksumUnavailable = errors.New("checksum file not found")

// allowHTTPInFetchURL relaxes the HTTPS-only restriction in fetchURL.
// It must only be set to true in tests.
var allowHTTPInFetchURL bool

// allowedUpdateHosts is the set of domains the update client may contact.
// All others are rejected to prevent SSRF via a compromised or hijacked update source.
var allowedUpdateHosts = []string{
	"github.com",
	"objects.githubusercontent.com",
	"quarkrelease.blob.core.windows.net",
	// Added at test time via allowHTTPInFetchURL + local httptest servers
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

const binaryName = "quark"

var backupName = fmt.Sprintf("%s_backup", binaryName)
var extractedName = fmt.Sprintf("%s_extracted", binaryName)

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

// ── Self-update preflight (#1609) ───────────────────────────────────────────

// ErrSelfUpdateUnavailable reports that this process cannot replace its own
// binary, whatever the update source. Callers should surface it as-is: the
// wrapped message names the directory, the account, and the way out.
var ErrSelfUpdateUnavailable = errors.New("cannot self-update")

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

// CanSelfUpdate reports whether the running process can replace its own binary
// in place, without downloading anything.
//
// replaceSelf creates its temp file in the directory holding the executable so
// the final rename is atomic and on the same filesystem — that is correct and
// must not be traded for a temp file in /tmp. The consequence is that the
// account the service runs as needs write access to that directory. Under the
// packaged systemd unit it did not, so every update attempt downloaded and
// extracted the full binary once per source before failing on a raw
// "permission denied" from deep inside the updater (#1609).
//
// The check is a real create-and-remove rather than a permission-bit test, so
// it agrees with what replaceSelf will actually be allowed to do — read-only
// mounts, ACLs and immutable flags included.
func CanSelfUpdate() error {
	dir, err := selfUpdateDir()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSelfUpdateUnavailable, err)
	}
	return canSelfUpdateIn(dir)
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

// SelfUpdatableBinDir and LegacyBinPath describe the install layout that makes
// self-update possible. They live here rather than in internal/install so the
// preflight message can name the fix; internal/install is what implements it.
const (
	SelfUpdatableBinDir = "/opt/quark/bin"
	LegacyBinPath       = "/usr/local/bin/quark"
)

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
