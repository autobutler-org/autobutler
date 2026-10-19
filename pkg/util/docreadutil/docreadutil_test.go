package docreadutil

import (
	"reflect"
	"strings"
	"testing"
)

// TestDocxCommentIsLeftOut reads a document whose comment body lives in
// word/comments.xml. The sentences the comment marks stay; the comment's
// own text and author do not, and the reader says comments are not shown.
func TestDocxCommentIsLeftOut(t *testing.T) {
	body := `<w:p><w:commentRangeStart w:id="0"/>` +
		`<w:r><w:t xml:space="preserve">first sentence</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t xml:space="preserve">second sentence</w:t></w:r>` +
		`<w:commentRangeEnd w:id="0"/>` +
		`<w:r><w:commentReference w:id="0"/></w:r></w:p>`
	comments := `<w:comment w:id="0" w:author="Ann Writer" w:date="2026-10-19T00:00:00Z" w:initials="AW">` +
		`<w:p><w:r><w:t>Please hide this comment</w:t></w:r></w:p>` +
		`</w:comment>`
	d := testDoc{
		body: body,
		rels: `<Relationship Id="rC" Type="` + relNS + `/comments" Target="comments.xml"/>`,
		files: map[string]string{
			"word/comments.xml": `<?xml version="1.0" encoding="UTF-8"?><w:comments ` + testNamespaces + `>` + comments + `</w:comments>`,
		},
	}
	out, result, err := convert(t, d.zip(t), OutputNDJSON, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	s := parseStream(t, out)
	if got := texts(s.blocks); !reflect.DeepEqual(got, []string{"first sentence", "second sentence"}) {
		t.Fatalf("texts = %q", got)
	}
	if s.end.Words != 4 {
		t.Errorf("words = %d, want 4", s.end.Words)
	}
	if !reflect.DeepEqual(result.Warnings, []string{warnComments}) || !reflect.DeepEqual(s.end.Warnings, []string{warnComments}) {
		t.Errorf("warnings = %q (stream %q), want %q", result.Warnings, s.end.Warnings, []string{warnComments})
	}
	for _, secret := range []string{"Please hide this comment", "Ann Writer"} {
		if strings.Contains(out, secret) {
			t.Errorf("%q is in the document", secret)
		}
	}
}
