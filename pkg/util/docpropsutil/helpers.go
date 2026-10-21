package docpropsutil

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// formatForName picks the metadata layout a file extension uses.
func formatForName(name string) (family, bool) {
	switch strings.ToLower(path.Ext(name)) {
	case ".docx", ".docm", ".dotx", ".dotm", ".pptx", ".pptm", ".ppsx", ".potx",
		".xlsx", ".xlsm", ".xltx", ".xltm":
		return formatOOXML, true
	case ".odt", ".ott", ".odp", ".otp", ".ods", ".ots":
		return formatODF, true
	case ".rtf":
		return formatRTF, true
	}
	return 0, false
}

// openPackage opens a zip package with the metadata bounds. The package
// limits are the metadata cap: nothing but a small entry is ever read.
func openPackage(src io.ReaderAt, size int64) (*officezip.Package, error) {
	pkg, err := officezip.Open(src, size, officezip.Limits{
		MaxPartBytes:  MaxMetadataBytes,
		MaxTotalBytes: 2 * MaxMetadataBytes,
		MaxElements:   100_000,
	})
	return pkg, wrapPackageError(err)
}

// decodePart decodes one XML metadata part into v. A part the package lacks
// leaves v untouched and reports false: a document without metadata is not
// an error.
func decodePart(pkg *officezip.Package, name string, v any) (bool, error) {
	if !pkg.Has(name) {
		return false, nil
	}
	dec, err := pkg.XML(name)
	if err != nil {
		return false, wrapPackageError(err)
	}
	defer dec.Close()
	if err := dec.Decode(v); err != nil {
		return false, wrapPackageError(err)
	}
	return true, nil
}

// wrapPackageError maps officezip's and encoding/xml's errors onto this
// package's, so callers match one set.
func wrapPackageError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, officezip.ErrTooLarge):
		return fmt.Errorf("%w: %v", ErrTooLarge, err)
	case errors.Is(err, ErrMalformed), errors.Is(err, ErrTooLarge):
		return err
	}
	return fmt.Errorf("%w: %v", ErrMalformed, err)
}

// count parses a stored count; anything that is not a positive integer is
// absent.
func count(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// date returns an RFC 3339 form of a stored date when it parses as one, and
// the trimmed text otherwise.
func date(s string) string {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return s
}
