package officezip

import (
	"encoding/xml"
	"io"
)

// The end of a zip: the end-of-central-directory record and, for an archive
// past 65,535 entries or 4 GiB, the zip64 record a locator before it points at.
const (
	eocdSignature       = 0x06054b50
	eocdLength          = 22
	zip64LocatorSig     = 0x07064b50
	zip64LocatorLength  = 20
	zip64EOCDSignature  = 0x06064b50
	zip64EOCDLength     = 56
	maxZipCommentLength = 0xFFFF
)

// manifestPath is where an ODF package lists its entries.
const manifestPath = "META-INF/manifest.xml"

// budgetReader reads one part, counting what it reads against its own limit
// and the package's remaining budget. A byte past either is never handed
// out: the read that would need it fails instead, and says which was passed.
type budgetReader struct {
	rc    io.ReadCloser
	pkg   *Package
	name  string
	limit int64
	read  int64
}

// tokenCounter passes tokens through from a part's decoder and fails once
// the part holds more than maxElements elements or nests them deeper than
// maxDepth.
type tokenCounter struct {
	dec         *xml.Decoder
	name        string
	maxElements int
	maxDepth    int
	depth       int
	elements    int
	// err is the limit the part broke; every later read repeats it.
	err error
}

// xRelationships is an OOXML relationships part. Names are matched without
// their namespace, as every OOXML reader in the wild does.
type xRelationships struct {
	Rels []xRelationship `xml:"Relationship"`
}

// xRelationship is one <Relationship> of a relationships part.
type xRelationship struct {
	ID         string `xml:"Id,attr"`
	Type       string `xml:"Type,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr"`
}

// xManifest is an ODF META-INF/manifest.xml.
type xManifest struct {
	Entries []xManifestEntry `xml:"file-entry"`
}

// xManifestEntry is one <manifest:file-entry>.
type xManifestEntry struct {
	FullPath   string    `xml:"full-path,attr"`
	MediaType  string    `xml:"media-type,attr"`
	Encryption *struct{} `xml:"encryption-data"`
}
