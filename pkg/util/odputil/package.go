package odputil

import (
	"encoding/xml"
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
	var manifest []officezip.ManifestEntry
	if pkg.Has(manifestPart) {
		var err error
		if manifest, err = pkg.Manifest(); err != nil {
			return err
		}
	}
	kind := ""
	for _, e := range manifest {
		if strings.EqualFold(e.FullPath, contentPart) && e.Encrypted {
			return fmt.Errorf("%w: content.xml is password-protected", ErrEncrypted)
		}
		if e.FullPath == "/" {
			kind = e.MediaType
		}
	}
	if pkg.Has(mimePart) {
		var err error
		if kind, err = readMimetype(pkg); err != nil {
			return err
		}
	}
	if kind != "" && !presentationKind(kind) {
		return fmt.Errorf("%w: media type %s", ErrMalformed, kind)
	}
	if !pkg.Has(contentPart) {
		return fmt.Errorf("%w: missing content.xml", ErrMalformed)
	}
	return nil
}

// readMimetype reads the uncompressed media-type file. It is a short label;
// anything longer than a label is not one.
func readMimetype(pkg *officezip.Package) (string, error) {
	rc, err := pkg.Part(mimePart)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	buf := make([]byte, 129)
	n, err := io.ReadFull(rc, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	if n > 128 {
		return "", fmt.Errorf("%w: mimetype is not a presentation", ErrMalformed)
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

func presentationKind(kind string) bool {
	kind = strings.TrimSpace(kind)
	if i := strings.IndexByte(kind, ';'); i >= 0 {
		kind = strings.TrimSpace(kind[:i])
	}
	return kind == mimePresentation || kind == mimeTemplate
}

// readTitle is meta.xml's dc:title, or "" when it has none. A meta.xml that
// cannot be read has none: the title is not worth refusing a deck over.
func readTitle(pkg *officezip.Package) string {
	if !pkg.Has(metaPart) {
		return ""
	}
	dec, err := pkg.XML(metaPart)
	if err != nil {
		return ""
	}
	defer dec.Close()
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Space != nsDC || start.Name.Local != "title" {
			continue
		}
		budget := &nodeBudget{nodes: 64, bytes: maxTitleBytes, what: "the title is too long"}
		n, err := readNode(dec, start, budget)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(n.innerText())
	}
}

// readStyles reads styles.xml: its fonts, named and automatic styles, and
// master pages. A package without one has only content.xml's styles.
func (im *importer) readStyles() error {
	if !im.pkg.Has(stylesPart) {
		return nil
	}
	dec, err := im.pkg.XML(stylesPart)
	if err != nil {
		return err
	}
	defer dec.Close()
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return xmlError(err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Space != nsOffice {
			continue
		}
		switch start.Name.Local {
		case "document-styles":
			continue
		case "font-face-decls", "styles", "automatic-styles", "master-styles":
			n, err := readNode(dec, start, im.budget)
			if err != nil {
				return err
			}
			im.styles.add(n, scopeMaster)
		default:
			if err := dec.Skip(); err != nil {
				return xmlError(err)
			}
		}
	}
}
