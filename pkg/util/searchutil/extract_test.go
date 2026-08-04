package searchutil

import (
	"strings"
	"testing"
)

func TestExtractAbdoc(t *testing.T) {
	json := `{"ops":[{"insert":"Hello, "},{"insert":"world","attributes":{"bold":true}},{"insert":"\n"}]}`
	text, ok := ExtractText("notes.abdoc", []byte(json))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(text, "Hello, world") {
		t.Errorf("expected 'Hello, world' in text, got: %q", text)
	}
}

func TestExtractAbdoc_SkipsEmbeds(t *testing.T) {
	json := `{"ops":[{"insert":"before"},{"insert":{"image":"data:img"}},{"insert":"after"}]}`
	text, ok := ExtractText("doc.abdoc", []byte(json))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if strings.Contains(text, "image") {
		t.Errorf("embed object should not appear in extracted text, got: %q", text)
	}
	if !strings.Contains(text, "before") || !strings.Contains(text, "after") {
		t.Errorf("expected text around embed, got: %q", text)
	}
}

func TestExtractAbsheet(t *testing.T) {
	json := `{"tabs":[{"name":"Sheet1","data":{"rows":[["Alice","30"],["Bob","25"]]}}]}`
	text, ok := ExtractText("data.absheet", []byte(json))
	if !ok {
		t.Fatal("expected ok=true")
	}
	for _, want := range []string{"Sheet1", "Alice", "Bob", "30", "25"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in extracted text, got: %q", want, text)
		}
	}
}

func TestExtractText_PlainText(t *testing.T) {
	text, ok := ExtractText("notes.txt", []byte("hello world"))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if text != "hello world" {
		t.Errorf("expected raw text passthrough, got: %q", text)
	}
}

func TestExtractText_Unsupported(t *testing.T) {
	_, ok := ExtractText("image.jpg", []byte{0xff, 0xd8})
	if ok {
		t.Error("expected ok=false for unsupported type")
	}
}

func TestIsIndexable(t *testing.T) {
	indexable := []string{"doc.abdoc", "sheet.absheet", "notes.txt", "readme.md", "data.csv"}
	for _, f := range indexable {
		if !IsIndexable(f) {
			t.Errorf("expected %q to be indexable", f)
		}
	}
	notIndexable := []string{"photo.jpg", "video.mp4", "archive.zip", "font.ttf"}
	for _, f := range notIndexable {
		if IsIndexable(f) {
			t.Errorf("expected %q to NOT be indexable", f)
		}
	}
}
