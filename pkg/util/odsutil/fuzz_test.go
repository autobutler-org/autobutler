package odsutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

// FuzzConvertToQsheet reads arbitrary bytes as a spreadsheet package.
// Every failure is one of the package's own errors, or the input was a
// spreadsheet and the envelope is well-formed JSON with rectangular sheets.
func FuzzConvertToQsheet(f *testing.F) {
	f.Add(mustPackage(f, contentXML(sheet("S", row(strCell("a")+floatCell("1.5", "1.5"))))))
	f.Add(mustPackage(f, contentXML(sheet("S", row(strCell("A"))+
		`<table:table-row table:number-rows-repeated="1000000000"><table:table-cell table:number-columns-repeated="1000000000"/></table:table-row>`))))
	f.Add(mustPackage(f, contentXML(sheet("S", row(strCell("A")+`<table:table-cell table:number-columns-repeated="1000000000"/>`+strCell("B"))))))
	f.Add([]byte("PK\x05\x06"))
	f.Add([]byte("<office:document-content/>"))
	f.Fuzz(func(t *testing.T, data []byte) {
		assertConvert(t, data)
	})
}

// FuzzContentXML wraps arbitrary XML in a spreadsheet package, so the
// parser is what the mutator reaches.
func FuzzContentXML(f *testing.F) {
	f.Add(contentXML(sheet("S", row(strCell("a")))))
	f.Add(contentXML(sheet("S", `<table:table-row table:number-rows-repeated="999999999999"><table:table-cell/></table:table-row>`+row(strCell("a")))))
	f.Add(contentXML(sheet("S", row(`<table:table-cell><text:p>A<text:s text:c="999999999"/>B</text:p></table:table-cell>`))))
	f.Add("<not")
	f.Fuzz(func(t *testing.T, content string) {
		data, err := packageBytes(content, mimeSpreadsheet, "")
		if err != nil {
			t.Fatal(err)
		}
		assertConvert(t, data)
	})
}

func assertConvert(t *testing.T, data []byte) {
	t.Helper()
	var out bytes.Buffer
	_, err := ConvertToQsheet(ConvertToQsheetParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out})
	if err != nil {
		if !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrTooLarge) && !errors.Is(err, ErrEncrypted) {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	var doc qsheetDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.Bytes())
	}
	if len(doc.Tabs) == 0 {
		t.Fatal("a successful conversion wrote no sheet")
	}
	for _, tab := range doc.Tabs {
		if len(tab.Data.Rows) == 0 {
			t.Fatal("a sheet has no rows")
		}
		width := len(tab.Data.Rows[0])
		if width == 0 {
			t.Fatal("a row has no cells")
		}
		for _, row := range tab.Data.Rows {
			if len(row) != width {
				t.Fatalf("ragged row in %q", tab.Name)
			}
		}
	}
}
