package v0_photos_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_photos "github.com/autobutler-org/quark/internal/server/api/v0/photos"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// Every Photos scale bug so far assumed the loaded page was the library
// (#3491). These tests page through GET /photos over a generated library and
// pin the paging contract: every page is full but the last, every page claims
// the whole library as its total, and the pages together are the library,
// each photo once, in sort order.

// scaleTakenPerList is how many capture dates one sort=taken listing reads,
// photoutil's takenDatesPerList.
const scaleTakenPerList = 200

// scalePhoto is one photo of the generated library, as the manifest knows it.
type scalePhoto struct {
	relPath string
	name    string
	mtime   time.Time
	// taken is the EXIF capture date, zero for a photo without one.
	taken time.Time
}

// scaleLibrary generates a library of n photos. It is deterministic and full
// of the ties a real library has: photos added three to a second, burst
// pairs sharing a capture date, the same file name in two folders. One photo
// in five has no EXIF date, and the capture dates run across 53 months in an
// order unrelated to the order the photos were added.
func scaleLibrary(n int) []scalePhoto {
	added := time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC)
	photos := make([]scalePhoto, n)
	for i := range photos {
		name := fmt.Sprintf("IMG_%05d.jpg", i)
		if i%10 == 0 {
			// Photos i and i+10 share a name, in different folders.
			name = fmt.Sprintf("dup_%04d.jpg", i/20)
		}
		p := scalePhoto{
			relPath: fmt.Sprintf("album_%02d/%s", i%17, name),
			name:    name,
			mtime:   added.Add(time.Duration(i/3) * time.Second),
		}
		if i%5 != 4 {
			burst := i / 2
			month := (burst * 7) % 53
			p.taken = time.Date(2019, time.Month(1+month), 1+burst%28, 12, burst%60, 0, 0, time.Local)
		}
		photos[i] = p
	}
	return photos
}

// scaleHarness serves the photos routes over one real files directory holding
// a generated library and a migrated database, acting as the system.
type scaleHarness struct {
	engine  *gin.Engine
	library []scalePhoto
	byPath  map[string]scalePhoto
}

func newScaleHarness(t *testing.T, n int) scaleHarness {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	filesDir, err := storageutil.GetFilesDir()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()

	h := scaleHarness{library: scaleLibrary(n), byPath: make(map[string]scalePhoto, n)}
	for _, p := range h.library {
		h.byPath[p.relPath] = p
		content := plain
		if !p.taken.IsZero() {
			content = withExifDate(plain, p.taken)
		}
		full := filepath.Join(filesDir, filepath.FromSlash(p.relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(full, p.mtime, p.mtime); err != nil {
			t.Fatal(err)
		}
	}

	svc := storageutil.NewStorageService(systemDevice{})
	registry := vfs.NewRegistry()
	if err := registry.Register(vfs.Namespace{ID: "files"}, vfs.NewStorageServiceVFS(svc, "files")); err != nil {
		t.Fatal(err)
	}
	deps := deputil.NewDependencies().
		WithStorageService(svc).
		WithVFSRegistry(registry).
		WithDatabase(dbtest.NewDB(t))

	gin.SetMode(gin.TestMode)
	h.engine = gin.New()
	h.engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c = ctxutil.With(c, "principal", accessutil.System)
		c.Next()
	})
	serverutil.RegisterRouterWithGroup(h.engine.Group("/api/v0"), v0_photos.NewRouter())
	return h
}

// withExifDate is plain with an EXIF segment whose DateTimeOriginal is taken,
// written "2006:01:02 15:04:05" the way cameras write it.
func withExifDate(plain []byte, taken time.Time) []byte {
	// TIFF header; IFD0 holding only the pointer to the Exif IFD; the Exif IFD
	// holding only DateTimeOriginal (ASCII); then the string.
	const ifd0, exifIFD, dateOffset = 8, 8 + 18, 8 + 18 + 18
	value := append([]byte(taken.Format("2006:01:02 15:04:05")), 0)
	tiff := new(bytes.Buffer)
	for _, v := range []any{
		[]byte("II"), uint16(42), uint32(ifd0),
		uint16(1), uint16(0x8769), uint16(4), uint32(1), uint32(exifIFD), uint32(0),
		uint16(1), uint16(0x9003), uint16(2), uint32(len(value)), uint32(dateOffset), uint32(0),
		value,
	} {
		_ = binary.Write(tiff, binary.LittleEndian, v)
	}
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	segment := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(payload)+2))
	segment = append(segment, payload...)
	// After the SOI marker, before everything else.
	return append(append(append([]byte{}, plain[:2]...), segment...), plain[2:]...)
}

// list fetches one page of GET /photos.
func (h scaleHarness) list(t *testing.T, query string, offset, limit int) v0_photos.PaginatedPhotosResponse {
	t.Helper()
	path := fmt.Sprintf("/api/v0/photos?%s&offset=%d&limit=%d", query, offset, limit)
	w := httptest.NewRecorder()
	h.engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s = %d: %s", path, w.Code, w.Body.String())
	}
	var page v0_photos.PaginatedPhotosResponse
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return page
}

// settleTakenDates lists sort=taken until every capture date has been read,
// the way the client lists again while pendingDates is above zero. Each
// listing reads its full allowance, so the count falls by that much each time
// and reaches zero within a bounded number of listings.
func (h scaleHarness) settleTakenDates(t *testing.T) {
	t.Helper()
	n := len(h.library)
	for listing := 1; ; listing++ {
		page := h.list(t, "sort=taken", 0, 1)
		if want := max(n-listing*scaleTakenPerList, 0); page.PendingDates != want {
			t.Fatalf("listing %d left %d dates pending, want %d", listing, page.PendingDates, want)
		}
		if page.PendingDates == 0 {
			return
		}
	}
}

// sortKey orders the manifest the way sort orders the library: by date added,
// by capture date standing in the date added where there is none, or by name
// ignoring case. Photos with equal keys may come in any order.
func sortKey(sort string, p scalePhoto) string {
	switch sort {
	case "name":
		return strings.ToLower(p.name)
	case "taken":
		if !p.taken.IsZero() {
			return fmt.Sprintf("%020d", p.taken.Unix())
		}
	}
	return fmt.Sprintf("%020d", p.mtime.Unix())
}

// assertPagesCoverLibrary pages through query limit at a time and checks the
// paging contract against the manifest.
func (h scaleHarness) assertPagesCoverLibrary(t *testing.T, sort, order string, limit int) {
	t.Helper()
	n := len(h.library)
	query := "sort=" + sort + "&order=" + order
	seen := make(map[string]bool, n)
	previous := ""
	pages := 0
	for offset := 0; offset < n; offset += limit {
		page := h.list(t, query, offset, limit)
		pages++
		if page.Total != n {
			t.Errorf("%s limit %d offset %d: total %d, want %d", query, limit, offset, page.Total, n)
		}
		if want := min(limit, n-offset); len(page.Photos) != want {
			t.Errorf("%s limit %d offset %d: %d photos, want %d", query, limit, offset, len(page.Photos), want)
		}
		for _, got := range page.Photos {
			p, ok := h.byPath[got.RelPath]
			switch {
			case !ok:
				t.Errorf("%s limit %d: unexpected %s", query, limit, got.RelPath)
				continue
			case seen[got.RelPath]:
				t.Errorf("%s limit %d: %s appears twice", query, limit, got.RelPath)
			}
			seen[got.RelPath] = true
			if sort == "taken" && got.TakenAt != takenUnix(p) {
				t.Errorf("%s: %s takenAt %d, want %d", query, got.RelPath, got.TakenAt, takenUnix(p))
			}
			key := sortKey(sort, p)
			if previous != "" && (order == "asc" && key < previous || order == "desc" && key > previous) {
				t.Errorf("%s limit %d: %s (%s) is out of order after %s", query, limit, got.RelPath, key, previous)
			}
			previous = key
		}
	}
	if want := (n + limit - 1) / limit; pages != want {
		t.Errorf("%s limit %d: %d pages, want %d", query, limit, pages, want)
	}
	for _, p := range h.library {
		if !seen[p.relPath] {
			t.Errorf("%s limit %d: missing %s", query, limit, p.relPath)
		}
	}

	// One past the end is an empty page that still knows the library's size.
	if past := h.list(t, query, n, limit); len(past.Photos) != 0 || past.Total != n {
		t.Errorf("%s limit %d past the end: %d photos, total %d; want none of %d", query, limit, len(past.Photos), past.Total, n)
	}
}

// takenUnix is the takenAt GET /photos reports for p under sort=taken.
func takenUnix(p scalePhoto) int64 {
	if p.taken.IsZero() {
		return 0
	}
	return p.taken.Unix()
}

// runScaleInvariants checks every sort, order and page size the grid asks
// for over a library of n photos. 75 does not divide the library, so its last
// page is short.
func runScaleInvariants(t *testing.T, n int) {
	h := newScaleHarness(t, n)
	h.settleTakenDates(t)
	for _, sort := range []string{"added", "taken", "name"} {
		for _, order := range []string{"desc", "asc"} {
			for _, limit := range []int{50, 75, 200} {
				t.Run(fmt.Sprintf("%s_%s_%d", sort, order, limit), func(t *testing.T) {
					h.assertPagesCoverLibrary(t, sort, order, limit)
				})
			}
		}
	}
}

func TestPhotoScale_PagesCoverTheLibraryExactly(t *testing.T) {
	runScaleInvariants(t, 1000)
}

// The full 10,000-photo library runs nightly, not on every pull request: set
// QUARK_PHOTOS_SCALE_10K=1 to run it.
func TestPhotoScale_PagesCoverTheLibraryExactly10k(t *testing.T) {
	if testing.Short() || os.Getenv("QUARK_PHOTOS_SCALE_10K") == "" {
		t.Skip("set QUARK_PHOTOS_SCALE_10K=1 to page through a 10,000-photo library")
	}
	runScaleInvariants(t, 10000)
}
