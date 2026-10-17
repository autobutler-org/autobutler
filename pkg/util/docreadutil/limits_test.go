package docreadutil

// cspell:ignore abstractNum abstractNumId sdt sdtContent styleId tbl tc tr

import (
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// allocated reports how many bytes fn allocates on the heap.
func allocated(fn func()) uint64 {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func TestBlockLimitTruncates(t *testing.T) {
	s, result := readStream(t, testDoc{body: strings.Repeat(para("word"), 100)}, Limits{MaxBlocks: 10})
	if len(s.blocks) != 10 || !s.end.Truncated || !result.Truncated || result.Blocks != 10 || s.end.Words != 10 {
		t.Fatalf("blocks %d, end %+v, result %+v", len(s.blocks), s.end, result)
	}
}

func TestOutputLimitTruncatesALongParagraph(t *testing.T) {
	const limit = 64 << 10
	body := "<w:p>" + strings.Repeat(`<w:r><w:t>`+strings.Repeat("x", 1000)+`</w:t></w:r>`, 1000) + "</w:p>" + para("after")
	out, result, err := convert(t, testDoc{body: body}.zip(t), OutputNDJSON, Limits{MaxOutputBytes: limit})
	if err != nil {
		t.Fatal(err)
	}
	s := parseStream(t, out)
	if !s.end.Truncated || !result.Truncated || len(s.blocks) != 1 {
		t.Fatalf("end %+v, %d blocks", s.end, len(s.blocks))
	}
	if len(out) > limit+1024 {
		t.Errorf("wrote %d bytes past a %d-byte limit", len(out), limit)
	}
	if n := len(plain(s.blocks[0])); n < limit/2 {
		t.Errorf("the truncated paragraph kept only %d bytes", n)
	}
}

func TestOutputLimitInsideATableKeepsWhatWasRead(t *testing.T) {
	const limit = 8 << 10
	row := `<w:tr>` + cell("", "a cell with some words in it") + cell("", "and another") + `</w:tr>`
	body := para("before") + `<w:tbl>` + strings.Repeat(row, 1000) + `</w:tbl>` + para("after")
	for _, output := range []Output{OutputNDJSON, OutputDelta} {
		out, result, err := convert(t, testDoc{body: body}.zip(t), output, Limits{MaxOutputBytes: limit})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Truncated || len(out) > 2*limit {
			t.Fatalf("output %d: truncated %v, %d bytes", output, result.Truncated, len(out))
		}
		if output != OutputNDJSON {
			continue
		}
		s := parseStream(t, out)
		if len(s.blocks) != 2 || s.blocks[1].Type != BlockTable {
			t.Fatalf("blocks = %v", texts(s.blocks))
		}
		if rows := len(s.blocks[1].Rows); rows == 0 || rows >= 1000 {
			t.Errorf("the truncated table has %d rows", rows)
		}
	}
}

func TestHeldPartsAreBounded(t *testing.T) {
	cases := map[string]testDoc{
		"styles past their byte cap": {styles: `<w:style w:styleId="Big"><w:name w:val="` + strings.Repeat("x", 5000) + `"/></w:style>`},
		"too many styles":            {styles: strings.Repeat(`<w:style w:styleId="s"/>`, 11)},
		"too many lists":             {numbering: strings.Repeat(`<w:abstractNum w:abstractNumId="1"/>`, 6) + strings.Repeat(`<w:num w:numId="1"/>`, 5)},
		"a deeply nested style part": {styles: strings.Repeat("<w:x>", 300) + strings.Repeat("</w:x>", 300)},
	}
	for name, d := range cases {
		d.body = para("x")
		out, _, err := convert(t, d.zip(t), OutputNDJSON, Limits{MaxHeldBytes: 4096, MaxDefinitions: 10})
		if !errors.Is(err, ErrTooLarge) {
			t.Errorf("%s: err = %v, want ErrTooLarge", name, err)
		}
		if out != "" {
			t.Errorf("%s: wrote %q before failing", name, out)
		}
	}
}

func TestPackageLimitsApply(t *testing.T) {
	_, _, err := convert(t, testDoc{body: para("x")}.zip(t), OutputNDJSON, Limits{Zip: officezip.Limits{MaxEntries: 2}})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestNestingBombFails(t *testing.T) {
	body := "<w:p>" + strings.Repeat("<w:sdt><w:sdtContent>", 100_000) + strings.Repeat("</w:sdtContent></w:sdt>", 100_000) + "</w:p>"
	data := testDoc{body: body}.zip(t)
	var err error
	if n := allocated(func() { _, _, err = convert(t, data, OutputNDJSON, Limits{}) }); n > 8<<20 {
		t.Errorf("a nesting bomb allocated %d bytes", n)
	}
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestReadingStopsAtTheLimit(t *testing.T) {
	// 200,000 paragraphs compress to almost nothing; reaching the block
	// limit must stop the read, not decompress the rest.
	data := testDoc{body: strings.Repeat(para("word"), 200_000)}.zip(t)
	var result ConvertResult
	n := allocated(func() {
		result, _ = Convert(ConvertParams{Source: bytes.NewReader(data), Size: int64(len(data)), Format: FormatDOCX,
			Out: io.Discard, Limits: Limits{MaxBlocks: 100}})
	})
	if !result.Truncated || result.Blocks != 100 {
		t.Fatalf("result = %+v", result)
	}
	if n > 4<<20 {
		t.Errorf("reading 100 blocks allocated %d bytes", n)
	}
}

func TestContextEndsTheConversion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data := testDoc{body: para("x")}.zip(t)
	_, err := Convert(ConvertParams{Ctx: ctx, Source: bytes.NewReader(data), Size: int64(len(data)), Format: FormatDOCX, Out: io.Discard})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// failingWriter fails every write.
type failingWriter struct{}

var errWrite = errors.New("the client went away")

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestWriteErrorsAreReturnedAsTheyAre(t *testing.T) {
	data := testDoc{body: para("x")}.zip(t)
	for _, output := range []Output{OutputNDJSON, OutputDelta} {
		_, err := Convert(ConvertParams{Source: bytes.NewReader(data), Size: int64(len(data)), Format: FormatDOCX, Output: output, Out: failingWriter{}})
		if !errors.Is(err, errWrite) || errors.Is(err, ErrMalformed) {
			t.Errorf("output %d: err = %v, want the write error", output, err)
		}
	}
}

func TestMalformedDocuments(t *testing.T) {
	good := testDoc{body: para("x")}.zip(t)
	cases := map[string][]byte{
		"empty":      {},
		"not a zip":  []byte("this is plain text, not a document"),
		"no body":    rezip(t, good, func(name string) bool { return name != "word/document.xml" }),
		"wrong root": testDocWithPart(t, "word/document.xml", `<p:presentation xmlns:p="urn:x"/>`),
		"bad styles": testDoc{body: para("x"), styles: `<w:style><w:name></w:style>`}.zip(t),
		"bad rels":   testDocWithPart(t, "word/_rels/document.xml.rels", `<Relationships><Relationship`),
		"bad xml":    testDocWithPart(t, "word/document.xml", `<w:document xmlns:w="urn:w"><w:body><w:p><w:r><w:t>x</w:r></w:p></w:body></w:document>`),
		"unclosed":   testDocWithPart(t, "word/document.xml", `<w:document xmlns:w="urn:w"><w:body><w:p>`),
		"not utf-8":  testDocWithPart(t, "word/document.xml", `<?xml version="1.0" encoding="ISO-8859-1"?><w:document xmlns:w="urn:w"/>`),
	}
	for name, data := range cases {
		out, _, err := convert(t, data, OutputNDJSON, Limits{})
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: err = %v, want ErrMalformed", name, err)
		}
		if out != "" {
			t.Errorf("%s: wrote %q before failing", name, out)
		}
	}
}

func TestAFailureMidwayLeavesNoEndLine(t *testing.T) {
	body := strings.Repeat(para("fine"), 3) + `<w:p><w:r><w:t>broken</w:x></w:r></w:p>`
	out, _, err := convert(t, testDoc{body: body}.zip(t), OutputNDJSON, Limits{})
	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
	if strings.Contains(out, `"type":"end"`) {
		t.Errorf("a stream that failed has an end line:\n%s", out)
	}
}

func TestABadTitleIsNoTitle(t *testing.T) {
	s, _ := readStream(t, testDoc{body: para("x"), title: "<unclosed"}, Limits{})
	if s.meta.Title != "" || len(s.blocks) != 1 {
		t.Errorf("stream = %+v", s)
	}
}

func TestConvertRefusesWhatItCannotRead(t *testing.T) {
	data := testDoc{body: para("x")}.zip(t)
	if _, err := Convert(ConvertParams{Source: bytes.NewReader(data), Size: int64(len(data)), Format: "odt", Out: io.Discard}); !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("odt: err = %v, want ErrUnsupportedFormat", err)
	}
	if _, err := Convert(ConvertParams{Format: FormatDOCX, Out: io.Discard}); err == nil {
		t.Error("no Source: want an error")
	}
}

// testDocWithPart is a small package with one part replaced.
func testDocWithPart(t *testing.T, name, body string) []byte {
	t.Helper()
	d := testDoc{body: para("x")}
	data := rezip(t, d.zip(t), func(n string) bool { return n != name })
	return appendEntry(t, data, name, body)
}

// appendEntry copies a zip and adds one entry to it.
func appendEntry(t testing.TB, data []byte, name, body string) []byte {
	t.Helper()
	return rezipWith(t, data, func(string) bool { return true }, name, body)
}
