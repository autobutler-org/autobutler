package officezip

import (
	"strings"
	"testing"
)

func TestResolveTarget(t *testing.T) {
	cases := []struct {
		source, target, want string
		ok                   bool
	}{
		{"word/document.xml", "media/image1.png", "word/media/image1.png", true},
		{"word/document.xml", "../customXml/item1.xml", "customXml/item1.xml", true},
		{"word/document.xml", "/word/styles.xml", "word/styles.xml", true},
		{"", "word/document.xml", "word/document.xml", true},
		{"word/document.xml", "media/a%20b.png", "word/media/a b.png", true},
		{"word/document.xml", "../../etc/passwd", "", false},
		{"word/document.xml", "..", "", false},
		{"word/document.xml", "", "", false},
		{"word/document.xml", `media\a.png`, "", false},
		{"word/document.xml", "%2e%2e/%2e%2e/x", "", false},
	}
	for _, c := range cases {
		got, ok := resolveTarget(c.source, c.target)
		if got != c.want || ok != c.ok {
			t.Errorf("resolveTarget(%q, %q) = %q, %v; want %q, %v", c.source, c.target, got, ok, c.want, c.ok)
		}
	}
}

func TestRelsPath(t *testing.T) {
	for part, want := range map[string]string{
		"":                  "_rels/.rels",
		"word/document.xml": "word/_rels/document.xml.rels",
		"content.xml":       "_rels/content.xml.rels",
	} {
		if got := relsPath(part); got != want {
			t.Errorf("relsPath(%q) = %q, want %q", part, got, want)
		}
	}
}

// FuzzResolveTarget checks that from a safe part name no target, however
// written, resolves to a name that climbs out of the package.
func FuzzResolveTarget(f *testing.F) {
	f.Add("word/document.xml", "media/image1.png")
	f.Add("word/document.xml", "../../x")
	f.Add("", "/a/../../b")
	f.Add("a/b/c.xml", "%2e%2e/%2e%2e/%2e%2e/x")
	f.Fuzz(func(t *testing.T, source, target string) {
		if source != "" && !safePartName(source) {
			// Sources are entry names, which Open has already checked.
			return
		}
		got, ok := resolveTarget(source, target)
		if !ok {
			if got != "" {
				t.Fatalf("unresolved target returned %q", got)
			}
			return
		}
		if strings.HasPrefix(got, "/") || strings.ContainsAny(got, "\\\x00") {
			t.Fatalf("resolveTarget(%q, %q) = %q", source, target, got)
		}
		for _, seg := range strings.Split(got, "/") {
			if seg == "" || seg == "." || seg == ".." {
				t.Fatalf("resolveTarget(%q, %q) = %q holds segment %q", source, target, got, seg)
			}
		}
	})
}
