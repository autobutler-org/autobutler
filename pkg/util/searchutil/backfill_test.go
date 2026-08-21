package searchutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- .abdoc / .absheet extraction ---

func TestIsIndexable_QuarkFormats(t *testing.T) {
	for _, path := range []string{"notes.abdoc", "budget.absheet", "NOTES.ABDOC"} {
		if !IsIndexable(path) {
			t.Errorf("IsIndexable(%q) = false, want true", path)
		}
	}
}

// A doc named "something.txt.abdoc" must be treated as a doc, not as text:
// filepath.Ext only sees the final extension.
func TestIsIndexable_DoubleExtensionUsesLast(t *testing.T) {
	if !IsIndexable("something.txt.abdoc") {
		t.Error("IsIndexable(\"something.txt.abdoc\") = false, want true")
	}
}

func TestExtractText_Abdoc(t *testing.T) {
	const delta = `{"ops":[{"insert":"hello world"},{"insert":"\nsecond line\n"}]}`
	path := writeTemp(t, "notes.abdoc", delta)

	got := ExtractText(path)
	const want = "hello world\nsecond line"
	if got != want {
		t.Errorf("ExtractText = %q, want %q", got, want)
	}
	// The JSON envelope must not leak into the index.
	if containsAny(got, `{"ops"`, `"insert"`) {
		t.Errorf("ExtractText leaked JSON syntax: %q", got)
	}
}

// An op's insert is an object for embeds (images and the like). Those carry no
// indexable text and must not break extraction of the surrounding prose.
func TestExtractText_AbdocSkipsEmbeds(t *testing.T) {
	const delta = `{"ops":[{"insert":"before "},{"insert":{"image":"a.png"}},{"insert":"after"}]}`
	path := writeTemp(t, "embed.abdoc", delta)

	if got, want := ExtractText(path), "before after"; got != want {
		t.Errorf("ExtractText = %q, want %q", got, want)
	}
}

func TestExtractText_Absheet(t *testing.T) {
	const sheet = `{"tabs":[{"name":"Budget","data":{"rows":[["=B1+B2","rent"],["",42]]}}]}`
	path := writeTemp(t, "budget.absheet", sheet)

	got := ExtractText(path)
	for _, want := range []string{"Budget", "=B1+B2", "rent", "42"} {
		if !containsAny(got, want) {
			t.Errorf("ExtractText = %q, missing %q", got, want)
		}
	}
	// Empty cells must not become stray tokens.
	if containsAny(got, `"rows"`, `"tabs"`) {
		t.Errorf("ExtractText leaked JSON syntax: %q", got)
	}
}

// Malformed JSON still gets indexed verbatim rather than dropped — a truncated
// or hand-edited document should stay searchable.
func TestExtractText_MalformedAbdocFallsBackToRaw(t *testing.T) {
	const broken = `{"ops":[{"insert":"unterminated`
	path := writeTemp(t, "broken.abdoc", broken)

	if got := ExtractText(path); got != broken {
		t.Errorf("ExtractText = %q, want raw fallback %q", got, broken)
	}
}

// The end-to-end case behind #1339: the query string lives only inside an
// .abdoc envelope, so it is findable only if the Delta is unwrapped.
func TestBackfillThenSearch_Abdoc(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "something.txt.abdoc"),
		`{"ops":[{"insert":"sdfsdfsadfsdffsdfasdf\n"}]}`)

	res, err := BackfillTree(context.Background(), db, "", dir)
	if err != nil {
		t.Fatalf("BackfillTree: %v", err)
	}
	if res.Indexed != 1 {
		t.Fatalf("Indexed = %d, want 1 (result %+v)", res.Indexed, res)
	}

	hits, err := Search(context.Background(), db, "sdfsdfsadfsdffsdfasdf", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1", len(hits))
	}
	if hits[0].RelPath != "something.txt.abdoc" {
		t.Errorf("RelPath = %q, want %q", hits[0].RelPath, "something.txt.abdoc")
	}
}

// --- BackfillTree ---

func TestBackfillTree_IndexesNestedAndSkipsBinary(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "top.txt"), "alpha")
	writeAt(t, filepath.Join(dir, "sub", "nested.md"), "bravo")
	writeAt(t, filepath.Join(dir, "photo.jpg"), "not indexable")

	res, err := BackfillTree(context.Background(), db, "SERIAL1", dir)
	if err != nil {
		t.Fatalf("BackfillTree: %v", err)
	}
	if res.Scanned != 3 {
		t.Errorf("Scanned = %d, want 3", res.Scanned)
	}
	if res.Indexed != 2 {
		t.Errorf("Indexed = %d, want 2", res.Indexed)
	}

	// Nested files must be stored with a forward-slash relative path so they
	// match the paths carried on file events.
	hits, err := Search(context.Background(), db, "bravo", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1", len(hits))
	}
	if hits[0].RelPath != "sub/nested.md" {
		t.Errorf("RelPath = %q, want %q", hits[0].RelPath, "sub/nested.md")
	}
	if hits[0].Serial != "SERIAL1" {
		t.Errorf("Serial = %q, want %q", hits[0].Serial, "SERIAL1")
	}
}

// Trashed files stay on disk but must never appear in search results.
func TestBackfillTree_SkipsTrash(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "kept.txt"), "charlie")
	writeAt(t, filepath.Join(dir, trashDirName, "deleted.txt"), "charlie")

	res, err := BackfillTree(context.Background(), db, "", dir)
	if err != nil {
		t.Fatalf("BackfillTree: %v", err)
	}
	if res.Indexed != 1 {
		t.Errorf("Indexed = %d, want 1", res.Indexed)
	}

	hits, err := Search(context.Background(), db, "charlie", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1 (trashed copy must be skipped)", len(hits))
	}
	if hits[0].RelPath != "kept.txt" {
		t.Errorf("RelPath = %q, want %q", hits[0].RelPath, "kept.txt")
	}
}

// Re-running must refresh existing entries rather than duplicate them, since
// the pass runs on every startup.
func TestBackfillTree_RerunUpdatesInPlace(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	writeAt(t, path, "original")

	if _, err := BackfillTree(context.Background(), db, "", dir); err != nil {
		t.Fatalf("first BackfillTree: %v", err)
	}
	writeAt(t, path, "revised")
	if _, err := BackfillTree(context.Background(), db, "", dir); err != nil {
		t.Fatalf("second BackfillTree: %v", err)
	}

	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM file_content`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Errorf("file_content has %d rows, want 1", rows)
	}

	hits, err := Search(context.Background(), db, "revised", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Errorf("got %d hits for updated content, want 1", len(hits))
	}
	stale, err := Search(context.Background(), db, "original", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("got %d hits for replaced content, want 0", len(stale))
	}
}

func TestBackfillTree_MissingDirIsNotAnError(t *testing.T) {
	db := newTestDB(t)
	res, err := BackfillTree(context.Background(), db, "", filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Errorf("BackfillTree on missing dir returned %v, want nil", err)
	}
	if res.Indexed != 0 {
		t.Errorf("Indexed = %d, want 0", res.Indexed)
	}
}

// --- helpers ---

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	writeAt(t, path, content)
	return path
}

func writeAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}
