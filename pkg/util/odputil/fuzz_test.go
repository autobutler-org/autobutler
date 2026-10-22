package odputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

// FuzzImportOdp reads arbitrary bytes as a presentation package. Every
// failure is one of the package's own errors, or the input was a
// presentation and the .qslide is well-formed JSON.
func FuzzImportOdp(f *testing.F) {
	f.Add(fixture{content: contentXML("", page(`<draw:frame svg:width="1cm" svg:height="1cm"><draw:text-box><text:p>a</text:p></draw:text-box></draw:frame>`))}.bytes(f))
	f.Add([]byte("PK\x05\x06"))
	f.Add([]byte("<office:document-content/>"))
	f.Fuzz(func(t *testing.T, data []byte) {
		assertImport(t, data)
	})
}

// FuzzContentXML wraps arbitrary XML in a presentation package, so the
// parser is what the mutator reaches.
func FuzzContentXML(f *testing.F) {
	f.Add(contentXML("", page(`<draw:custom-shape svg:width="2cm" svg:height="1cm" draw:transform="rotate (1) translate (1cm 1cm)"><draw:enhanced-geometry draw:type="star5"/></draw:custom-shape>`)))
	f.Add(contentXML("", page(`<draw:frame svg:width="1cm" svg:height="1cm"><table:table><table:table-row table:number-rows-repeated="999999999"><table:table-cell/></table:table-row></table:table></draw:frame>`)))
	f.Add(contentXML("", page(`<draw:g><draw:g><draw:line svg:x1="0cm" svg:y1="0cm" svg:x2="1cm" svg:y2="1cm"/></draw:g></draw:g>`)))
	f.Add(contentXML("", page(`<draw:polygon svg:width="1cm" svg:height="1cm" svg:viewBox="0 0 0 0" draw:points="1,1"/>`)))
	f.Add("<not")
	f.Fuzz(func(t *testing.T, content string) {
		assertImport(t, fixture{content: content}.bytes(t))
	})
}

func assertImport(t *testing.T, data []byte) {
	t.Helper()
	var out bytes.Buffer
	store := func(_, _ string, r io.Reader) (string, error) {
		_, err := io.Copy(io.Discard, r)
		return "part:x", err
	}
	_, err := ImportOdp(ImportOdpParams{Source: bytes.NewReader(data), Size: int64(len(data)), Out: &out, StoreMedia: store})
	if err != nil {
		if !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrTooLarge) && !errors.Is(err, ErrEncrypted) {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	var d deck
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.Bytes())
	}
}
