// Package officezip opens the zip packages office documents are made of —
// DOCX, PPTX, XLSX and their ODF cousins ODT, ODP and ODS — with every read
// bounded, so a hostile file costs one failed request rather than the
// server's memory.
//
// A package is opened in place from an [io.ReaderAt] and its size: [Open]
// checks the entry count and the size of the zip's directory against
// [Limits] before archive/zip reads a byte of it, and refuses entry names
// that climb out of the package or that two entries share. A part is then
// read with [Package.Part], which never hands out more than
// [Limits.MaxPartBytes] whatever size the entry declares, and every part
// read from one package draws on a shared [Limits.MaxTotalBytes] budget.
// Reaching either cap is [ErrTooLarge], never a silent truncation.
//
// XML parts are read with [Package.XML], a token decoder that fails once a
// part holds more than [Limits.MaxElements] elements or nests them deeper
// than [Limits.MaxDepth]. encoding/xml resolves no external entities and
// expands no entity a document declares, so a part cannot grow past what it
// holds.
//
// [Package.Relationships] reads an OOXML part's relationships with external
// targets marked, so a reader never follows one, and [Package.Manifest] reads
// an ODF package's META-INF/manifest.xml.
//
// A Package is meant for one request on one goroutine, and is not safe for
// concurrent use.
package officezip

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Default limits, the bounds the read-only document viewers start from
// (docs/architecture/read-only-documents.md, "Memory and streaming").
const (
	// DefaultMaxEntries bounds the entries of one package, checked against
	// the zip's directory before the directory is read.
	DefaultMaxEntries = 20_000
	// DefaultMaxDirectoryBytes bounds the zip directory, which archive/zip
	// reads whole before anything else.
	DefaultMaxDirectoryBytes = 16 << 20 // 16 MiB
	// DefaultMaxPartBytes bounds one part unpacked.
	DefaultMaxPartBytes = 64 << 20 // 64 MiB
	// DefaultMaxTotalBytes bounds what one package unpacks to, every part
	// read from it together.
	DefaultMaxTotalBytes = 256 << 20 // 256 MiB
	// DefaultMaxElements bounds the elements of one XML part.
	DefaultMaxElements = 2_000_000
	// DefaultMaxDepth bounds how deeply one XML part's elements nest.
	DefaultMaxDepth = 256
)

var (
	// ErrTooLarge is a package or a part past one of its [Limits].
	ErrTooLarge = errors.New("officezip: the document is too large")
	// ErrMalformed is a file that is not a well-formed office package: not
	// a zip, a zip with an unsafe or duplicated entry name, a part that
	// fails to decompress, or malformed XML.
	ErrMalformed = errors.New("officezip: the document is malformed")
	// ErrMissingPart is a part the package does not hold.
	ErrMissingPart = errors.New("officezip: missing part")
)

// Limits bounds what opening and reading a package may cost. A zero field
// takes its Default value, so Limits{} is the default set.
type Limits struct {
	// MaxEntries bounds the zip's entry count.
	MaxEntries int
	// MaxDirectoryBytes bounds the zip's central directory.
	MaxDirectoryBytes int64
	// MaxPartBytes bounds one part unpacked.
	MaxPartBytes int64
	// MaxTotalBytes bounds every part read from the package together.
	MaxTotalBytes int64
	// MaxElements bounds the elements of one XML part.
	MaxElements int
	// MaxDepth bounds how deeply one XML part's elements nest.
	MaxDepth int
}

// Package is an opened office zip: its entries by name and what is left of
// its read budget. Part names are matched ignoring case, since OPC part
// names are case-insensitive; a package with two entries whose names differ
// only by case is refused when it is opened.
type Package struct {
	limits Limits
	// files is every entry in directory order, and parts the same entries
	// by lowercased name.
	files []*zip.File
	parts map[string]*zip.File
	// remaining is how many uncompressed bytes may still be read, over
	// every part together.
	remaining int64
}

// Decoder is an XML token decoder over one part that counts what it reads.
// Every way of reading through it — Token, RawToken, Decode, DecodeElement,
// Skip — fails with [ErrTooLarge] once the part holds more than
// [Limits.MaxElements] elements or nests deeper than [Limits.MaxDepth], and
// with [ErrMalformed] on malformed XML. As with [xml.Decoder.Token], the
// bytes of a CharData, Comment, ProcInst or Directive token are valid only
// until the next token is read.
//
// Close it when done; it closes the part.
type Decoder struct {
	*xml.Decoder
	counter *tokenCounter
	part    io.Closer
}

// Relationship is one OOXML relationship of a part.
type Relationship struct {
	// ID is the relationship's id, the r:id a part refers to it by.
	ID string
	// Type is the relationship type URI.
	Type string
	// Target is the target exactly as the relationship wrote it.
	Target string
	// External reports TargetMode="External": a URL or a path outside the
	// package, which a reader must never fetch or follow.
	External bool
	// Part is the internal target resolved to a part name. It is empty for
	// an external target, and for one that climbs out of the package.
	Part string
}

// ManifestEntry is one file-entry of an ODF package's manifest.
type ManifestEntry struct {
	// FullPath is the entry's path as the manifest wrote it; "/" is the
	// package itself.
	FullPath string
	// MediaType is the entry's declared media type.
	MediaType string
	// Encrypted reports that the entry carries encryption-data: its bytes
	// cannot be read without the document's password.
	Encrypted bool
}

// Open opens the office package held by src, size bytes long. The entry
// count and the size of the zip's directory are checked against limits
// before the directory is read; an entry whose name could climb out of the
// package, or that two entries share ignoring case, refuses the package with
// [ErrMalformed].
func Open(src io.ReaderAt, size int64, limits Limits) (*Package, error) {
	limits = limits.withDefaults()
	if size <= 0 {
		return nil, fmt.Errorf("%w: the file is empty", ErrMalformed)
	}
	entries, dirSize, err := zipDirectory(src, size)
	if err != nil {
		return nil, err
	}
	if entries > uint64(limits.MaxEntries) {
		return nil, fmt.Errorf("%w: the package holds more than %d entries", ErrTooLarge, limits.MaxEntries)
	}
	if dirSize > uint64(limits.MaxDirectoryBytes) {
		return nil, fmt.Errorf("%w: the package's directory is larger than %d bytes", ErrTooLarge, limits.MaxDirectoryBytes)
	}
	zr, err := zip.NewReader(src, size)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return nil, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if len(zr.File) > limits.MaxEntries {
		return nil, fmt.Errorf("%w: the package holds more than %d entries", ErrTooLarge, limits.MaxEntries)
	}
	p := &Package{
		limits:    limits,
		files:     zr.File,
		parts:     make(map[string]*zip.File, len(zr.File)),
		remaining: limits.MaxTotalBytes,
	}
	for _, f := range zr.File {
		if !safePartName(f.Name) {
			return nil, fmt.Errorf("%w: entry %q names a path outside the package", ErrMalformed, f.Name)
		}
		key := strings.ToLower(f.Name)
		if _, dup := p.parts[key]; dup {
			return nil, fmt.Errorf("%w: entry %q appears twice", ErrMalformed, f.Name)
		}
		p.parts[key] = f
	}
	return p, nil
}

// Names returns every entry name in the package, in directory order.
func (p *Package) Names() []string {
	names := make([]string, len(p.files))
	for i, f := range p.files {
		names[i] = f.Name
	}
	return names
}

// Has reports whether the package holds the part, ignoring case.
func (p *Package) Has(name string) bool {
	_, ok := p.parts[strings.ToLower(name)]
	return ok
}

// Part opens a part for reading. A part that declares more than
// [Limits.MaxPartBytes], or more than is left of the package's budget, is
// refused before a byte is decompressed; one that expands past either while
// it is read fails that read with [ErrTooLarge]. A part the package does not
// hold is [ErrMissingPart].
func (p *Package) Part(name string) (io.ReadCloser, error) {
	f, ok := p.parts[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingPart, name)
	}
	if f.UncompressedSize64 > uint64(p.limits.MaxPartBytes) {
		return nil, fmt.Errorf("%w: %s declares more than %d bytes", ErrTooLarge, name, p.limits.MaxPartBytes)
	}
	if f.UncompressedSize64 > uint64(max(p.remaining, 0)) {
		return nil, fmt.Errorf("%w: the package unpacks to more than %d bytes", ErrTooLarge, p.limits.MaxTotalBytes)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrMalformed, name, err)
	}
	return &budgetReader{rc: rc, pkg: p, name: name, limit: p.limits.MaxPartBytes}, nil
}

// XML opens an XML part as a counting token decoder, in strict mode. Close
// the decoder when done.
func (p *Package) XML(name string) (*Decoder, error) {
	rc, err := p.Part(name)
	if err != nil {
		return nil, err
	}
	counter := &tokenCounter{
		dec:         xml.NewDecoder(rc),
		name:        name,
		maxElements: p.limits.MaxElements,
		maxDepth:    p.limits.MaxDepth,
	}
	return &Decoder{Decoder: xml.NewTokenDecoder(counter), counter: counter, part: rc}, nil
}

// Relationships reads the relationships of an OOXML part from its _rels
// part; the empty name reads the package's own, _rels/.rels. A part with no
// relationships part has none, which is not an error. Each internal target
// is resolved to a part name; an external one is marked and never resolved.
func (p *Package) Relationships(part string) ([]Relationship, error) {
	name := relsPath(part)
	if !p.Has(name) {
		return nil, nil
	}
	var doc xRelationships
	if err := p.decodeXML(name, &doc); err != nil {
		return nil, err
	}
	rels := make([]Relationship, 0, len(doc.Rels))
	for _, r := range doc.Rels {
		rel := Relationship{ID: r.ID, Type: r.Type, Target: r.Target, External: strings.EqualFold(r.TargetMode, "External")}
		if !rel.External {
			rel.Part, _ = resolveTarget(part, r.Target)
		}
		rels = append(rels, rel)
	}
	return rels, nil
}

// Manifest reads an ODF package's META-INF/manifest.xml. A package without
// one is [ErrMissingPart].
func (p *Package) Manifest() ([]ManifestEntry, error) {
	var doc xManifest
	if err := p.decodeXML(manifestPath, &doc); err != nil {
		return nil, err
	}
	entries := make([]ManifestEntry, 0, len(doc.Entries))
	for _, e := range doc.Entries {
		entries = append(entries, ManifestEntry{FullPath: e.FullPath, MediaType: e.MediaType, Encrypted: e.Encryption != nil})
	}
	return entries, nil
}

// Elements returns how many elements the decoder has read so far.
func (d *Decoder) Elements() int { return d.counter.elements }

// Close closes the part the decoder reads.
func (d *Decoder) Close() error { return d.part.Close() }
