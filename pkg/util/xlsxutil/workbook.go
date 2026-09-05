package xlsxutil

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

// officeDocumentRel is the relationship type naming the package's main part,
// which for a spreadsheet is the workbook.
const officeDocumentRel = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"

// worksheetRel is the relationship type naming one worksheet of a workbook.
const worksheetRel = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet"

// relationshipNS is the namespace a <sheet>'s r:id attribute is qualified by.
// Matched on the attribute's namespace rather than its "r:" prefix, which a
// writer is free to name something else.
const relationshipNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

// defaultWorkbookPart is where every writer in practice puts the workbook. It
// is the fallback for a package whose root relationships do not name one.
const defaultWorkbookPart = "xl/workbook.xml"

// excelEpoch is the day serial 0 denotes under the 1900 date system. It is
// 1899-12-30 rather than 1899-12-31 because Excel counts a 29th of February
// 1900 that never happened, and every serial after it inherits the extra day.
var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

// excelEpoch1904 is the same for a workbook saved with the 1904 date system,
// which has no phantom leap day to correct for.
var excelEpoch1904 = time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)

// readWorkbook resolves the package: it indexes the parts, finds the workbook,
// and pairs each worksheet's tab name with the part holding its rows.
func readWorkbook(zr *zip.Reader) (workbook, error) {
	parts := indexParts(zr)

	workbookPart := findWorkbookPart(parts)
	if workbookPart == "" {
		return workbook{}, fmt.Errorf("%w: no workbook part", ErrNotSpreadsheet)
	}
	wb, ok := parts[workbookPart]
	if !ok {
		return workbook{}, fmt.Errorf("%w: workbook part %s is missing", ErrNotSpreadsheet, workbookPart)
	}

	names, date1904, err := readSheetNames(wb)
	if err != nil {
		return workbook{}, err
	}
	if len(names) == 0 {
		return workbook{}, fmt.Errorf("%w: workbook declares no sheets", ErrNotSpreadsheet)
	}
	if len(names) > MaxSheets {
		return workbook{}, fmt.Errorf("%w: %d sheets exceeds the maximum of %d", ErrTooLarge, len(names), MaxSheets)
	}

	targets, err := readRelationships(parts, relsPartFor(workbookPart))
	if err != nil {
		return workbook{}, err
	}

	epoch := excelEpoch
	if date1904 {
		epoch = excelEpoch1904
	}

	book := workbook{parts: parts, epoch: epoch}
	baseDir := path.Dir(workbookPart)
	for _, s := range names {
		rel, ok := targets[s.relID]
		if !ok || rel.Type != worksheetRel {
			// A <sheet> whose relationship is absent or points at something
			// that is not a worksheet — a chartsheet, say — has no rows to
			// convert. Skipping it keeps the rest of the workbook readable.
			continue
		}
		part, ok := parts[resolveTarget(baseDir, rel.Target)]
		if !ok {
			continue
		}
		book.sheets = append(book.sheets, sheetRef{name: s.name, part: part})
	}
	if len(book.sheets) == 0 {
		return workbook{}, fmt.Errorf("%w: workbook has no readable worksheets", ErrNotSpreadsheet)
	}
	return book, nil
}

// indexParts maps every archive entry to its package-relative name. Zip stores
// forward-slash paths, and a writer may or may not have anchored them at the
// root, so both spellings normalize to the same key.
func indexParts(zr *zip.Reader) partIndex {
	parts := make(partIndex, len(zr.File))
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		parts[normalizePart(f.Name)] = f
	}
	return parts
}

// normalizePart is the key a part is looked up by: cleaned, forward-slashed,
// and not anchored at the root.
func normalizePart(name string) string {
	return strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(name, "\\", "/")), "/")
}

// relsPartFor is where a part's relationships live: "_rels/<name>.rels" beside
// the part itself.
func relsPartFor(part string) string {
	return path.Join(path.Dir(part), "_rels", path.Base(part)+".rels")
}

// resolveTarget resolves a relationship target against the directory of the
// part that declared it. A target anchored at the root is already absolute.
func resolveTarget(baseDir, target string) string {
	target = strings.ReplaceAll(target, "\\", "/")
	if strings.HasPrefix(target, "/") {
		return normalizePart(target)
	}
	return normalizePart(path.Join(baseDir, target))
}

// findWorkbookPart reads the package's root relationships for the part marked
// as the office document. A package that does not name one falls back to where
// every writer puts it.
func findWorkbookPart(parts partIndex) string {
	rels, err := readRelationships(parts, "_rels/.rels")
	if err == nil {
		for _, rel := range rels {
			if rel.Type == officeDocumentRel {
				if target := resolveTarget(".", rel.Target); parts[target] != nil {
					return target
				}
			}
		}
	}
	if parts[defaultWorkbookPart] != nil {
		return defaultWorkbookPart
	}
	return ""
}

// readRelationships parses one .rels part into its entries by ID. A package
// with no such part has no relationships, which is not an error here — the
// caller decides whether what it was looking for was required.
func readRelationships(parts partIndex, relsPart string) (map[string]relationship, error) {
	rels := make(map[string]relationship)
	f, ok := parts[relsPart]
	if !ok {
		return rels, nil
	}
	rc, err := openPart(f, MaxPartBytes)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, partError(relsPart, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}
		var rel relationship
		if err := dec.DecodeElement(&rel, &start); err != nil {
			return nil, partError(relsPart, err)
		}
		if rel.ID != "" {
			rels[rel.ID] = rel
		}
	}
	return rels, nil
}

// sheetName pairs a tab's name with the relationship naming the part that
// holds its rows.
type sheetName struct {
	name  string
	relID string
}

// readSheetNames streams the workbook part for its <sheet> entries, in the
// order they are declared, and for whether the workbook counts dates from
// 1904. The attributes are read off the element rather than decoded into a
// struct because r:id is namespace-qualified, and matching it by namespace is
// what keeps a writer's choice of prefix from mattering.
func readSheetNames(f *zip.File) ([]sheetName, bool, error) {
	rc, err := openPart(f, MaxPartBytes)
	if err != nil {
		return nil, false, err
	}
	defer rc.Close()

	var (
		sheets   []sheetName
		date1904 bool
	)
	dec := xml.NewDecoder(rc)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, false, partError(f.Name, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "workbookPr":
			for _, attr := range start.Attr {
				if attr.Name.Local == "date1904" && (attr.Value == "1" || attr.Value == "true") {
					date1904 = true
				}
			}
		case "sheet":
			var s sheetName
			for _, attr := range start.Attr {
				switch {
				case attr.Name.Local == "name" && attr.Name.Space == "":
					s.name = attr.Value
				case attr.Name.Local == "id" && attr.Name.Space == relationshipNS:
					s.relID = attr.Value
				}
			}
			if s.relID != "" {
				sheets = append(sheets, s)
			}
		}
	}
	return sheets, date1904, nil
}
