// Package docpropsutil reads the properties an office document carries about
// itself — title, author, dates, and the page, word and character counts its
// authoring application last saved — without parsing the document's body.
// OOXML (DOCX, PPTX, XLSX and their macro-enabled kin) keeps them in
// docProps/core.xml and docProps/app.xml, ODF (ODT, ODP, ODS) in meta.xml, and
// RTF in the \info group at the head of the file.
//
// Only that small metadata entry is read: a package is opened in place with
// officezip's bounds, the entry is capped at [MaxMetadataBytes] whatever size
// it declares, and an RTF is scanned only as far as [MaxRTFHeaderBytes]. The
// counts are what the application stored, never recounted
// (docs/architecture/read-only-documents.md, "Page and word counts, document
// properties").
package docpropsutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

const (
	// MaxMetadataBytes bounds one metadata entry unpacked. Real ones are a
	// few kilobytes; the cap keeps a hostile entry from costing more.
	MaxMetadataBytes = 1 << 20 // 1 MiB
	// MaxRTFHeaderBytes bounds how much of an RTF is scanned for its \info
	// group, which writers put before the body.
	MaxRTFHeaderBytes = 1 << 20 // 1 MiB
	// CountsSourceSaved is [Properties.CountsSource]: the counts are what
	// the authoring application last stored.
	CountsSourceSaved = "saved"
)

// Properties is what a document says about itself. A field the document
// does not carry is its zero value and is left out of the JSON.
type Properties struct {
	Title   string `json:"title,omitempty"`
	Author  string `json:"author,omitempty"`
	Subject string `json:"subject,omitempty"`
	// Created and Modified are RFC 3339 when the document's date parses as
	// one, and the document's own text otherwise.
	Created     string `json:"created,omitempty"`
	Modified    string `json:"modified,omitempty"`
	Pages       int    `json:"pages,omitempty"`
	Words       int    `json:"words,omitempty"`
	Characters  int    `json:"characters,omitempty"`
	Slides      int    `json:"slides,omitempty"`
	Sheets      int    `json:"sheets,omitempty"`
	Application string `json:"application,omitempty"`
	// CountsSource is always [CountsSourceSaved].
	CountsSource string `json:"countsSource"`
}

// ErrUnsupportedFormat is a file name whose extension is no format this
// package reads.
var ErrUnsupportedFormat = errors.New("docpropsutil: unsupported document format")

// ErrMalformed is a file that is not the format its name says, or whose
// metadata entry is not well-formed.
var ErrMalformed = errors.New("docpropsutil: the document is malformed")

// ErrTooLarge is a metadata entry past [MaxMetadataBytes] or the package
// limits.
var ErrTooLarge = errors.New("docpropsutil: the document's metadata is too large")

// ExtractParams is a document already opened for random access.
type ExtractParams struct {
	// Source is the document, read in place.
	Source io.ReaderAt
	// Size is the document's length in bytes.
	Size int64
	// Name is the document's file name; its extension picks the format.
	Name string
}

// ExtractResult is what a document says about itself.
type ExtractResult struct {
	Properties Properties
}

// Extract reads the properties of one document. Metadata the document does
// not carry is not an error: the result is simply empty. A file that is not
// what its name says is [ErrMalformed], and an oversize metadata entry is
// [ErrTooLarge].
func Extract(params ExtractParams) (ExtractResult, error) {
	format, ok := formatForName(params.Name)
	if !ok {
		return ExtractResult{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, filepath.Base(params.Name))
	}
	var (
		props Properties
		err   error
	)
	switch format {
	case formatOOXML:
		props, err = readOOXML(params.Source, params.Size)
	case formatODF:
		props, err = readODF(params.Source, params.Size, strings.ToLower(path.Ext(params.Name)))
	default:
		props, err = readRTF(io.NewSectionReader(params.Source, 0, params.Size))
	}
	if err != nil {
		return ExtractResult{}, err
	}
	props.CountsSource = CountsSourceSaved
	return ExtractResult{Properties: props}, nil
}

// ReadParams says which document's properties to read.
type ReadParams struct {
	// Ctx bounds opening the document.
	Ctx context.Context
	// Registry holds the namespace of the device Serial names.
	Registry vfs.Registry
	// FilePath is the files-relative path of the document.
	FilePath string
	// Serial identifies the device, empty for the internal one.
	Serial string
}

// ReadResult is a document's properties.
type ReadResult struct {
	Properties Properties
}

// Read opens a document through the device's namespace and extracts its
// properties. A missing file is a [fileutil.NotFoundError]; a name no format
// reads, or a file that is not what its name says, is a
// [fileutil.UnsupportedError].
func Read(params ReadParams) (ReadResult, error) {
	if _, ok := formatForName(params.FilePath); !ok {
		return ReadResult{}, &fileutil.UnsupportedError{
			Err: fmt.Errorf("not a document Quark can read properties from: %s", filepath.Base(params.FilePath)),
		}
	}
	fsys, err := fileutil.FilesVFS(params.Registry, params.Serial)
	if err != nil {
		return ReadResult{}, err
	}
	f, err := fsys.Open(params.Ctx, params.FilePath)
	if errors.Is(err, vfs.ErrIsDirectory) {
		return ReadResult{}, &fileutil.UnsupportedError{Err: fmt.Errorf("not a file: %s", params.FilePath)}
	}
	if err != nil {
		return ReadResult{}, &fileutil.NotFoundError{Err: err}
	}
	defer f.Close()
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return ReadResult{}, err
	}
	out, err := Extract(ExtractParams{Source: f, Size: size, Name: params.FilePath})
	if errors.Is(err, ErrMalformed) || errors.Is(err, ErrTooLarge) || errors.Is(err, ErrUnsupportedFormat) {
		return ReadResult{}, &fileutil.UnsupportedError{Err: err}
	}
	if err != nil {
		return ReadResult{}, err
	}
	return ReadResult(out), nil
}
