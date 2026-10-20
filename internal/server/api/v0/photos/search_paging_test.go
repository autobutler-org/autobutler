package v0_photos_test

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The file search the Photos box calls stops at 500 names and the response
// has no next page and no total (#3523). The library listing already pages,
// so a name search on it has to return every match, with each page's total
// equal to the match count. This is the shared 1,000-photo library plus
// enough extra beach photos to make 600 matches.
func TestPhotoSearch_PagesSixHundredMatches(t *testing.T) {
	h := newSearchHarness(t)
	const extra = 350
	if got := len(libraryWhere(func(i int) bool { return i%4 == 0 })) + extra; got != 600 {
		t.Fatalf("fixture has %d beach matches, want 600", got)
	}
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := range extra {
		h.write(t, fmt.Sprintf("shared/beach_extra_%04d.jpg", i), start.Add(time.Duration(i)*time.Minute))
	}
	// A folder named beach does not make the photo inside it a match, and a
	// text file whose name matches is not a photo.
	h.write(t, "beach/plain.jpg", start)
	h.write(t, "shared/beach_notes.txt", start)

	want := libraryWhere(func(i int) bool { return i%4 == 0 })
	for i := range extra {
		want = append(want, fmt.Sprintf("shared/beach_extra_%04d.jpg", i))
	}
	slices.SortFunc(want, func(a, b string) int {
		return strings.Compare(strings.ToLower(filepath.Base(a)), strings.ToLower(filepath.Base(b)))
	})

	for _, query := range []string{"q=beach", "q=BEACH"} {
		got, totals := h.listAll(t, query+"&sort=name&order=asc", 200)
		if !slices.Equal(got, want) {
			sameSet(t, query, got, want)
			t.Errorf("%s: pages are not in name order, or a page was cut off", query)
		}
		if len(totals) < 2 {
			t.Errorf("%s: got %d page(s), want more than one so the total is what says there is more", query, len(totals))
		}
		for _, total := range totals {
			if total != 600 {
				t.Errorf("%s: a page claimed a total of %d, want 600", query, total)
			}
		}
	}

	// A blank search is the whole library, not an empty answer and not the
	// beach matches alone. The text file is not a photo.
	const images = searchLibrarySize + extra + 1
	got, totals := h.listAll(t, "q=%20%20&sort=name&order=asc", 200)
	if len(got) != images {
		t.Errorf("blank q returned %d photos, want %d", len(got), images)
	}
	for _, total := range totals {
		if total != images {
			t.Errorf("blank q: a page claimed a total of %d, want %d", total, images)
		}
	}
	if slices.Contains(got, "shared/beach_notes.txt") {
		t.Error("blank q included beach_notes.txt, which is not a photo")
	}
	if !slices.Contains(got, "beach/plain.jpg") {
		t.Error("blank q left out beach/plain.jpg")
	}
}
