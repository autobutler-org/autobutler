package docpropsutil

import (
	"io"
	"strings"
)

// readODF reads meta.xml. ext picks what meta:page-count means: slides in a
// presentation, pages in a text document, nothing in a spreadsheet, where
// meta:table-count is the sheet count.
func readODF(src io.ReaderAt, size int64, ext string) (Properties, error) {
	pkg, err := openPackage(src, size)
	if err != nil {
		return Properties{}, err
	}
	var doc struct {
		Meta xODFMeta `xml:"meta"`
	}
	if _, err := decodePart(pkg, "meta.xml", &doc); err != nil {
		return Properties{}, err
	}
	m := doc.Meta
	author := strings.TrimSpace(m.InitialCreator)
	if author == "" {
		author = strings.TrimSpace(m.Creator)
	}
	props := Properties{
		Title:       strings.TrimSpace(m.Title),
		Author:      author,
		Subject:     strings.TrimSpace(m.Subject),
		Created:     date(m.Created),
		Modified:    date(m.Modified),
		Application: strings.TrimSpace(m.Generator),
		Words:       count(m.Statistic.Words),
		Characters:  count(m.Statistic.Characters),
	}
	switch ext {
	case ".odp", ".otp":
		props.Slides = count(m.Statistic.Pages)
	case ".ods", ".ots":
		props.Sheets = count(m.Statistic.Tables)
	default:
		props.Pages = count(m.Statistic.Pages)
	}
	return props, nil
}
