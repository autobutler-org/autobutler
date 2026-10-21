package docpropsutil

import (
	"io"
	"strings"
)

// readOOXML reads docProps/core.xml and docProps/app.xml.
func readOOXML(src io.ReaderAt, size int64) (Properties, error) {
	pkg, err := openPackage(src, size)
	if err != nil {
		return Properties{}, err
	}
	var core xCore
	if _, err := decodePart(pkg, "docProps/core.xml", &core); err != nil {
		return Properties{}, err
	}
	var app xApp
	if _, err := decodePart(pkg, "docProps/app.xml", &app); err != nil {
		return Properties{}, err
	}
	props := Properties{
		Title:       strings.TrimSpace(core.Title),
		Author:      strings.TrimSpace(core.Creator),
		Subject:     strings.TrimSpace(core.Subject),
		Created:     date(core.Created),
		Modified:    date(core.Modified),
		Application: strings.TrimSpace(app.Application),
		Pages:       count(app.Pages),
		Words:       count(app.Words),
		Characters:  count(app.Characters),
		Slides:      count(app.Slides),
	}
	vs := app.HeadingPairs.Variants
	for i := 0; i+1 < len(vs); i++ {
		if strings.TrimSpace(vs[i].Name) == "Worksheets" {
			props.Sheets = count(vs[i+1].Count)
		}
	}
	return props, nil
}
