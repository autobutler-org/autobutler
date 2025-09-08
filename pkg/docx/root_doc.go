package docx

import (
	_ "embed"
	"os"
	"path/filepath"
	"sync"

	"autobutler/pkg/docx/types"
	"github.com/nbio/xml"
)

// RootDoc represents the root document of an Office Open XML (OOXML) document.
// It contains information about the document path, file map, the document structure,
// and relationships with other parts of the document.
type RootDoc struct {
	Path string // Path represents the path of the document.
	// TODO: Need to actually parse out some of the file map, specifically word/numbering.xml
	FileMap     sync.Map      `xmls:"-"` // FileMap is a synchronized map for managing files related to the document.
	RootRels    Relationships // RootRels represents relationships at the root level.
	ContentType ContentTypes
	Document    *Document     // Document is the main document structure.
	Numbering   *Numbering    // Numbering format document
	DocStyles   *types.Styles // Document styles

	rID        int // rId is used to generate unique relationship IDs.
	ImageCount uint
}

type RootDocOptions struct{}

// NewRootDoc creates a new instance of the RootDoc structure.
func NewRootDoc(options RootDocOptions) *RootDoc {
	return &RootDoc{
		Numbering: NewNumbering(),
	}
}

//go:embed templates/default.docx
var defaultDocx []byte

// NewRootDocument creates a new document from the default template.
func NewRootDocument() (*RootDoc, error) {
	return Unpack(&defaultDocx)
}

// OpenDocument opens a document from the given file name.
func OpenDocument(fileName string) (*RootDoc, error) {
	docxContent, err := os.ReadFile(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}
	return Unpack(&docxContent)
}

// LoadDocXml decodes the provided XML data and returns a Document instance.
// It is used to load the main document structure from the document file.
//
// Parameters:
//   - fileName: The name of the document file.
//   - fileBytes: The XML data representing the main document structure.
//
// Returns:
//   - doc: The Document instance containing the decoded main document structure.
//   - err: An error, if any occurred during the decoding process.
func LoadDocXml(rd *RootDoc, fileName string, fileBytes []byte) (*Document, error) {
	doc := NewDocument(rd)
	err := xml.Unmarshal(fileBytes, doc)
	if err != nil {
		return nil, err
	}

	doc.RelativePath = fileName
	return doc, nil
}

func LoadNumberingXml(rd *RootDoc, fileName string, fileBytes []byte) (*Numbering, error) {
	numbering := NewNumbering()
	err := xml.Unmarshal(fileBytes, numbering)
	if err != nil {
		return nil, err
	}

	numbering.RelativePath = fileName
	return numbering, nil
}

// Load styles.xml into Styles struct
func LoadStyles(fileName string, fileBytes []byte) (*types.Styles, error) {
	styles := types.Styles{}
	err := xml.Unmarshal(fileBytes, &styles)
	if err != nil {
		return nil, err
	}

	styles.RelativePath = fileName
	return &styles, nil
}
