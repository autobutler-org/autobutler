package storageutil

import (
	"path/filepath"
	"sort"
	"strings"
)

// supportedExts is the set of archive extensions we accept for listing,
// reading and extraction. To add a new format: add its extension here.
// mholt/archiver handles the rest as long as the underlying Go library for
// that format is available; a zip under another name goes in zipExts too.
var supportedExts = map[string]struct{}{
	".zip": {},
	".rar": {},
	".7z":  {},
	".tar": {},
	// A compressed tar, under its long name and its short one.
	".tar.gz": {}, ".tgz": {},
	".tar.bz2": {}, ".tbz2": {}, ".tbz": {},
	".tar.xz": {}, ".txz": {},
	".tar.zst": {}, ".tzst": {},
	// A bare compressed stream: one file.
	".gz": {}, ".bz2": {}, ".xz": {}, ".zst": {},
	// Comic book archives.
	".cbz": {}, ".cb7": {},
	// Packages that are zips underneath: Java, Android, Python wheels.
	".jar": {}, ".apk": {}, ".whl": {},
}

// zipExts are the extensions of archives that are zips under another name.
// They are read in place as zips, never streamed.
var zipExts = map[string]struct{}{
	".zip": {}, ".cbz": {}, ".jar": {}, ".apk": {}, ".whl": {},
}

// IsZipArchive reports whether name is a zip, whatever it is called.
func IsZipArchive(name string) bool {
	_, ok := zipExts[archiveExt(name)]
	return ok
}

// SupportedArchiveExts returns the sorted list of archive extensions that can
// be extracted. Useful for UI hints and validation.
func SupportedArchiveExts() []string {
	exts := make([]string, 0, len(supportedExts))
	for ext := range supportedExts {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

// IsSupportedArchive reports whether name ends in an archive extension Quark
// can list, read and extract.
func IsSupportedArchive(name string) bool {
	_, ok := supportedExts[archiveExt(name)]
	return ok
}

// ArchiveStem is an archive's name without its archive extension, double
// extensions included: "foo.tar.gz" and "foo.zip" are both "foo". It names the
// folder an archive extracts into and the file a bare compressed stream
// decompresses to.
func ArchiveStem(name string) string {
	base := filepath.Base(name)
	stem := strings.TrimSuffix(base, base[len(base)-len(archiveExt(base)):])
	// Strip a trailing .tar that may remain after stripping .gz/.bz2/.xz/.zst.
	return strings.TrimSuffix(stem, ".tar")
}

// archiveExt returns the canonical extension for an archive path, handling
// double-extensions like ".tar.gz".
func archiveExt(path string) string {
	lower := strings.ToLower(path)
	for _, ext := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst"} {
		if strings.HasSuffix(lower, ext) {
			return ext
		}
	}
	if strings.HasSuffix(lower, ".tgz") {
		return ".tgz"
	}
	return strings.ToLower(filepath.Ext(path))
}
