package docx

import (
	"github.com/nbio/xml"
)

// This element specifies the contents of a main document part in a WordprocessingML document.
type Document struct {
	XMLName     xml.Name `xml:"w:document"`
	XMLNS       string   `xml:"xmlns,attr"`
	XMLNSa      string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:a,attr"`
	XMLNSc      string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:c,attr"`
	XMLNScr     string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:cr,attr"`
	XMLNSdgm    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:dgm,attr"`
	XMLNSlc     string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:lc,attr"`
	XMLNSm      string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:m,attr"`
	XMLNSmc     string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:mc,attr"`
	XMLNSo      string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:o,attr"`
	XMLNSpic    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:pic,attr"`
	XMLNSr      string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:r,attr"`
	XMLNSsl     string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:sl,attr"`
	XMLNSv      string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:v,attr"`
	XMLNSw10    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:w10,attr"`
	XMLNSw14    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:w14,attr"`
	XMLNSw15    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:w15,attr"`
	XMLNSw16    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:w16,attr"`
	XMLNSw16cex string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:w16cex,attr"`
	XMLNSw16cid string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:w16cid,attr"`
	XMLNSw      string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:w,attr"`
	XMLNSwne    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:wne,attr"`
	XMLNSwp     string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:wp,attr"`
	XMLNSwpg    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:wpg,attr"`
	XMLNSwps    string   `xml:"http://www.w3.org/2000/xmlns/ xmlns:wps,attr"`

	// Reference to the RootDoc
	Root *RootDoc `xml:"-"`

	// Elements
	Background *Background
	Body       *Body

	//lint:ignore SA5008 Ignore this field in serialization
	DocRels Relationships `xml:"-"`
	RID     int           `xml:"-"`

	RelativePath string `xml:"-"`
}

func NewDocument(root *RootDoc) *Document {
	return &Document{
		Root:        root,
		Background:  nil,
		Body:        NewBody(root),
		XMLNS:       "http://schemas.microsoft.com/office/tasks/2019/documenttasks",
		XMLNSa:      "http://schemas.openxmlformats.org/drawingml/2006/main",
		XMLNSc:      "http://schemas.openxmlformats.org/drawingml/2006/chart",
		XMLNScr:     "http://schemas.microsoft.com/office/comments/2020/reactions",
		XMLNSdgm:    "http://schemas.openxmlformats.org/drawingml/2006/diagram",
		XMLNSlc:     "http://schemas.openxmlformats.org/drawingml/2006/lockedCanvas",
		XMLNSm:      "http://schemas.openxmlformats.org/officeDocument/2006/math",
		XMLNSmc:     "http://schemas.openxmlformats.org/markup-compatibility/2006",
		XMLNSo:      "urn:schemas-microsoft-com:office:office",
		XMLNSpic:    "http://schemas.openxmlformats.org/drawingml/2006/picture",
		XMLNSr:      "http://schemas.openxmlformats.org/officeDocument/2006/relationships",
		XMLNSsl:     "http://schemas.openxmlformats.org/schemaLibrary/2006/main",
		XMLNSv:      "urn:schemas-microsoft-com:vml",
		XMLNSw10:    "urn:schemas-microsoft-com:office:word",
		XMLNSw14:    "http://schemas.microsoft.com/office/word/2010/wordml",
		XMLNSw15:    "http://schemas.microsoft.com/office/word/2012/wordml",
		XMLNSw16:    "http://schemas.microsoft.com/office/word/2018/wordml",
		XMLNSw16cex: "http://schemas.microsoft.com/office/word/2018/wordml/cex",
		XMLNSw16cid: "http://schemas.microsoft.com/office/word/2016/wordml/cid",
		XMLNSw:      "http://schemas.openxmlformats.org/wordprocessingml/2006/main",
		XMLNSwne:    "http://schemas.microsoft.com/office/word/2006/wordml",
		XMLNSwp:     "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing",
		XMLNSwpg:    "http://schemas.microsoft.com/office/word/2010/wordprocessingGroup",
		XMLNSwps:    "http://schemas.microsoft.com/office/word/2010/wordprocessingShape",
	}
}

// IncRelationID increments the relation ID of the document and returns the new ID.
// This method is used to generate unique IDs for relationships within the document.
func (doc *Document) IncRelationID() int {
	doc.RID += 1
	return doc.RID
}
