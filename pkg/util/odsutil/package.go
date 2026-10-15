package odsutil

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// checkPackage refuses a package that is encrypted or that declares itself
// to be some other kind of document. A package with no media type is judged
// by its content.xml.
func checkPackage(pkg *officezip.Package) error {
	encrypted, err := contentEncrypted(pkg)
	if err != nil {
		return err
	}
	if encrypted {
		return fmt.Errorf("%w: content.xml is password-protected", ErrEncrypted)
	}
	kind, err := packageKind(pkg)
	if err != nil {
		return err
	}
	if kind != "" && !spreadsheetKind(kind) {
		return fmt.Errorf("%w: media type %s", ErrMalformed, kind)
	}
	if !pkg.Has(contentPart) {
		return fmt.Errorf("%w: missing content.xml", ErrMalformed)
	}
	return nil
}

func contentEncrypted(pkg *officezip.Package) (bool, error) {
	if !pkg.Has("META-INF/manifest.xml") {
		return false, nil
	}
	entries, err := pkg.Manifest()
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if strings.EqualFold(e.FullPath, contentPart) && e.Encrypted {
			return true, nil
		}
	}
	return false, nil
}

func packageKind(pkg *officezip.Package) (string, error) {
	if pkg.Has(mimePart) {
		return readMimetype(pkg)
	}
	if !pkg.Has("META-INF/manifest.xml") {
		return "", nil
	}
	entries, err := pkg.Manifest()
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.FullPath == "/" {
			return strings.TrimSpace(e.MediaType), nil
		}
	}
	return "", nil
}

// readMimetype reads the uncompressed media-type file. It is a short label;
// anything longer than a label is not one.
func readMimetype(pkg *officezip.Package) (string, error) {
	rc, err := pkg.Part(mimePart)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	buf := make([]byte, 128)
	n, err := rc.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	var extra [1]byte
	n2, err2 := rc.Read(extra[:])
	if n2 > 0 {
		return "", fmt.Errorf("%w: mimetype is not a spreadsheet", ErrMalformed)
	}
	if err2 != nil && !errors.Is(err2, io.EOF) {
		return "", err2
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

func spreadsheetKind(kind string) bool {
	kind = strings.TrimSpace(kind)
	if i := strings.IndexByte(kind, ';'); i >= 0 {
		kind = strings.TrimSpace(kind[:i])
	}
	return kind == mimeSpreadsheet || kind == mimeTemplate
}
