package docpropsutil

// cspell:ignore dcterms

// family is which of the three metadata layouts a file extension uses.
type family int

const (
	formatOOXML family = iota
	formatODF
	formatRTF
)

// xCore is docProps/core.xml. Elements match on local name, so the dc:,
// dcterms: and cp: prefixes need no namespace plumbing.
type xCore struct {
	Title    string `xml:"title"`
	Creator  string `xml:"creator"`
	Subject  string `xml:"subject"`
	Created  string `xml:"created"`
	Modified string `xml:"modified"`
}

// xApp is docProps/app.xml.
type xApp struct {
	Application string `xml:"Application"`
	Pages       string `xml:"Pages"`
	Words       string `xml:"Words"`
	Characters  string `xml:"Characters"`
	Slides      string `xml:"Slides"`
	// HeadingPairs is a vector of alternating heading and count, which is
	// where a workbook says how many worksheets it holds.
	HeadingPairs struct {
		Variants []struct {
			Name  string `xml:"lpstr"`
			Count string `xml:"i4"`
		} `xml:"vector>variant"`
	} `xml:"HeadingPairs"`
}

// xODFMeta is meta.xml's office:meta element.
type xODFMeta struct {
	Title          string `xml:"title"`
	Subject        string `xml:"subject"`
	InitialCreator string `xml:"initial-creator"`
	Creator        string `xml:"creator"`
	Created        string `xml:"creation-date"`
	Modified       string `xml:"date"`
	Generator      string `xml:"generator"`
	Statistic      struct {
		Pages      string `xml:"page-count,attr"`
		Words      string `xml:"word-count,attr"`
		Characters string `xml:"character-count,attr"`
		Tables     string `xml:"table-count,attr"`
	} `xml:"document-statistic"`
}
