package photoutil_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// The tests in this file prove duplicate detection safe end to end on a
// generated library (#3490): files go into an in-memory device, the real
// backfill decodes them and stores their dHash and SHA-256, ListDuplicates
// groups what it stored, and the real delete path trashes what was marked.
// Every image is drawn here, deterministically, so nothing binary is
// committed and every expected group is written down next to its fixture.

// safetyScene describes one generated landscape: a sky gradient with clouds,
// a sun, a line of hills and, for a burst, a subject crossing the frame.
type safetyScene struct {
	// clouds seeds the cloud texture. Two photos of different skies differ
	// here; every variant of one photo shares it.
	clouds uint32
	// sunX is the sun's horizontal position, 0 to 1.
	sunX float64
	// hillPhase shifts the hills sideways.
	hillPhase float64
	// subjectX places a dark upright subject; negative leaves it out.
	subjectX float64
}

// sunset is the palette every scene shares: a hard negative is a photo
// that matches another's colors and differs in what is in it.
var sunsetTop, sunsetBottom = color.RGBA{250, 140, 60, 255}, color.RGBA{90, 40, 110, 255}

// render draws s at w x h.
func (s safetyScene) render(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		fy := float64(y) / float64(h)
		for x := range w {
			fx := float64(x) / float64(w)
			cloud := (cloudNoise(s.clouds, fx, fy) - 0.5) * 120
			c := color.RGBA{
				R: clamp8(lerp(sunsetTop.R, sunsetBottom.R, fy) + cloud),
				G: clamp8(lerp(sunsetTop.G, sunsetBottom.G, fy) + cloud),
				B: clamp8(lerp(sunsetTop.B, sunsetBottom.B, fy) + cloud),
				A: 255,
			}
			if math.Hypot(fx-s.sunX, (fy-0.5)*float64(h)/float64(w)) < 0.08 {
				c = color.RGBA{255, 230, 120, 255}
			}
			if fy > 0.62+0.06*math.Sin(fx*9+s.hillPhase) {
				c = color.RGBA{30, 20, 40, 255}
			}
			if s.subjectX >= 0 && math.Abs(fx-s.subjectX) < 0.04 && fy > 0.45 && fy < 0.75 {
				c = color.RGBA{10, 10, 10, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func lerp(a, b uint8, t float64) float64 { return float64(a)*(1-t) + float64(b)*t }

func clamp8(v float64) uint8 { return uint8(math.Max(0, math.Min(255, v))) }

// cloudNoise is bilinear value noise over a 7x5 lattice, coarse enough to
// survive the dHash's 9x8 averaging, as clouds do.
func cloudNoise(seed uint32, fx, fy float64) float64 {
	x, y := fx*7, fy*5
	i, j := int(x), int(y)
	tx, ty := x-float64(i), y-float64(j)
	top := latticeValue(seed, i, j)*(1-tx) + latticeValue(seed, i+1, j)*tx
	bottom := latticeValue(seed, i, j+1)*(1-tx) + latticeValue(seed, i+1, j+1)*tx
	return top*(1-ty) + bottom*ty
}

func latticeValue(seed uint32, i, j int) float64 {
	v := seed*2654435761 ^ uint32(i)*40503 ^ uint32(j)*2246822519
	v ^= v >> 13
	v *= 3266489917
	v ^= v >> 16
	return float64(v%1000) / 1000
}

func encodeJPEG(t *testing.T, img image.Image, quality int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// crop copies the part of img inside r, so it encodes on its own.
func crop(img image.Image, r image.Rectangle) image.Image {
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)
	return out
}

// rotatedWithTag stores img turned 90° counterclockwise, tagged with EXIF
// orientation 6 ("rotate 90° clockwise to display"), the way a phone held
// upright saves a photo. Upright, it is img again.
func rotatedWithTag(t *testing.T, img image.Image) []byte {
	t.Helper()
	b := img.Bounds()
	stored := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := range b.Dy() {
		for x := range b.Dx() {
			stored.Set(y, b.Dx()-1-x, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	body := encodeJPEG(t, stored, 90)

	var tiff bytes.Buffer
	tiff.WriteString("II")
	for _, v := range []any{
		uint16(42), uint32(8), // header, IFD0 offset
		uint16(1),                                       // one entry
		uint16(0x0112), uint16(3), uint32(1), uint16(6), // Orientation SHORT 6
		uint16(0), uint32(0), // padding, no next IFD
	} {
		_ = binary.Write(&tiff, binary.LittleEndian, v)
	}
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	app1 := binary.BigEndian.AppendUint16([]byte{0xFF, 0xE1}, uint16(len(payload)+2))
	out := append([]byte{}, body[:2]...)
	out = append(out, app1...)
	out = append(out, payload...)
	return append(out, body[2:]...)
}

// hashImage draws a 9x8-cell image whose dHash is exactly want: each row is
// a walk that steps down where a bit is set (left brighter than right) and
// up where it is not. It is how the threshold boundary is pinned to the bit.
func hashImage(want uint64) image.Image {
	const cell = 10
	img := image.NewGray(image.Rect(0, 0, 9*cell, 8*cell))
	for row := range 8 {
		v := 128
		for col := range 9 {
			if col > 0 {
				if want>>(row*8+col-1)&1 == 1 {
					v -= 12
				} else {
					v += 12
				}
			}
			for y := row * cell; y < (row+1)*cell; y++ {
				for x := col * cell; x < (col+1)*cell; x++ {
					img.SetGray(x, y, color.Gray{Y: uint8(v)})
				}
			}
		}
	}
	return img
}

// flipBits flips n distinct bits of h (n at most 11), six apart, so the
// flipped bits are spread across every row.
func flipBits(h uint64, n int) uint64 {
	for i := range n {
		h ^= 1 << (i * 6 % 64)
	}
	return h
}

// safetyTrashFS is a device namespace with a trash, as every real device
// has: DeleteFiles trashes through [vfs.Trasher]. Trash takes each file out
// of the namespace and keeps its bytes; a path in fail is refused, the way a
// read-only or failing disk refuses a rename, and stops the batch there.
type safetyTrashFS struct {
	*vfs.MemVFS
	vfs.Trasher // the rest of the trash API is never called here
	fail        map[string]bool
	trashed     map[string][]byte
}

func (f *safetyTrashFS) Trash(ctx context.Context, paths []string, opts vfs.TrashOptions) ([]vfs.TrashedItem, error) {
	var items []vfs.TrashedItem
	for _, p := range paths {
		rel := strings.TrimPrefix(path.Join("/", opts.RootDir, p), "/")
		if f.fail[rel] {
			return items, fmt.Errorf("trash %q: %w", rel, errors.New("input/output error"))
		}
		src, err := f.Open(ctx, rel)
		if errors.Is(err, vfs.ErrNotFound) {
			continue
		}
		if err != nil {
			return items, err
		}
		data, err := io.ReadAll(src)
		_ = src.Close()
		if err != nil {
			return items, err
		}
		if err := f.Delete(ctx, rel, vfs.DeleteOptions{}); err != nil {
			return items, err
		}
		name := fmt.Sprint(len(f.trashed) + 1)
		f.trashed[name] = data
		items = append(items, vfs.TrashedItem{OriginalPath: rel, TrashName: name})
	}
	return items, nil
}

// safetyLibrary is a generated photo library on the internal drive and one
// USB drive, indexed the way the server indexes one.
type safetyLibrary struct {
	ctx      context.Context
	database *db.DatabaseSqlc
	registry vfs.Registry
	devices  map[string]*safetyTrashFS
}

// newSafetyLibrary writes files, keyed serial then path, into a device each,
// and runs the hash backfill over them.
func newSafetyLibrary(t *testing.T, files map[string]map[string][]byte) *safetyLibrary {
	t.Helper()
	lib := &safetyLibrary{
		ctx:      context.Background(),
		database: dbtest.NewDB(t),
		registry: vfs.NewRegistry(),
		devices:  map[string]*safetyTrashFS{},
	}
	for serial, byPath := range files {
		fsys := &safetyTrashFS{
			MemVFS:  vfs.NewMemVFS(vfs.FilesNamespace(serial)),
			fail:    map[string]bool{},
			trashed: map[string][]byte{},
		}
		for p, data := range byPath {
			if err := fsys.Write(lib.ctx, p, bytes.NewReader(data), vfs.WriteOptions{}); err != nil {
				t.Fatal(err)
			}
		}
		if err := lib.registry.Register(vfs.Namespace{ID: vfs.FilesNamespace(serial)}, fsys); err != nil {
			t.Fatal(err)
		}
		lib.devices[serial] = fsys
	}
	result, err := photoutil.BackfillHashes(photoutil.BackfillHashesParams{
		Ctx: lib.ctx, Queries: lib.database.Queries, Registry: lib.registry,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 0 {
		t.Fatalf("backfill failed on %d photos", result.Failed)
	}
	return lib
}

// groups lists the duplicates at the default threshold, as the page asks for
// them, each as "kind: serial|path ..." in the order they came back.
func (lib *safetyLibrary) groups(t *testing.T) []photoutil.DuplicateGroup {
	t.Helper()
	result, err := photoutil.ListDuplicates(photoutil.ListDuplicatesParams{
		Ctx:       lib.ctx,
		Queries:   lib.database.Queries,
		Threshold: photoutil.ParseDuplicateThreshold(""),
		Access:    systemAccess(t),
		Exists:    photoutil.ExistsIn(lib.ctx, lib.registry),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Groups
}

func groupStrings(groups []photoutil.DuplicateGroup) []string {
	out := []string{}
	for _, g := range groups {
		s := g.Kind + ":"
		for _, p := range g.Photos {
			s += " " + p.DeviceSerial + "|" + p.RelPath
		}
		out = append(out, s)
	}
	return out
}

// storedHashes is every stored dHash, keyed "serial|path".
func (lib *safetyLibrary) storedHashes(t *testing.T) map[string]string {
	t.Helper()
	rows, err := lib.database.Queries.ListNearDuplicates(lib.ctx)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.DeviceSerial+"|"+r.RelPath] = r.Dhash.String
	}
	return out
}

// distance is the stored dHash distance between two "serial|path" keys.
func (lib *safetyLibrary) distance(t *testing.T, a, b string) int {
	t.Helper()
	hashes := lib.storedHashes(t)
	ha, okA := hashes[a]
	hb, okB := hashes[b]
	if !okA || !okB {
		t.Fatalf("no stored dHash for %q or %q", a, b)
	}
	return photoutil.HammingDistanceHex(ha, hb)
}

// readable reports whether a photo still opens and decodes on its device,
// and returns its SHA-256.
func (lib *safetyLibrary) readable(t *testing.T, serial, relPath string) (bool, [32]byte) {
	t.Helper()
	f, err := lib.devices[serial].Open(lib.ctx, relPath)
	if err != nil {
		return false, [32]byte{}
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, [32]byte{}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, [32]byte{}
	}
	if _, _, err := photoutil.DecodeImage(f); err != nil {
		return false, [32]byte{}
	}
	return true, [32]byte(h.Sum(nil))
}

// deleteCopies deletes paths on serial through the real delete path and
// waits for its background cleanup to drop their hashes.
func (lib *safetyLibrary) deleteCopies(t *testing.T, serial string, paths ...string) error {
	t.Helper()
	_, err := fileutil.DeleteFiles(fileutil.DeleteFilesParams{
		Registry:  lib.registry,
		EventBus:  eventbus.New(),
		Database:  lib.database,
		RootDir:   "/",
		FilePaths: paths,
		Serial:    serial,
	})
	if err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		hashes := lib.storedHashes(t)
		left := 0
		for _, p := range paths {
			if _, ok := hashes[serial+"|"+p]; ok {
				left++
			}
		}
		if left == 0 {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("hashes of %v were never dropped after the delete", paths)
	return nil
}

// TestDuplicatesSafety_Corpus groups a generated library with a manifest of
// every relationship in it and requires precision and recall of 1: every
// expected group found, nothing else grouped. Exact copies across names,
// folders and devices are one exact group; a file whose bytes differ is never
// in an exact group, even when its pixels are the same; resized,
// recompressed, rotated and lightly cropped copies are near; a heavy crop
// and sunsets in the same colors with different skies are not grouped.
func TestDuplicatesSafety_Corpus(t *testing.T) {
	beach := safetyScene{clouds: 1, sunX: 0.3, subjectX: -1}
	original := beach.render(320, 240)
	originalJPEG := encodeJPEG(t, original, 90)
	harbor := safetyScene{clouds: 7, sunX: 0.6, hillPhase: 1, subjectX: -1}.render(320, 240)
	harborJPEG := encodeJPEG(t, harbor, 90)

	lib := newSafetyLibrary(t, map[string]map[string][]byte{
		"": {
			"2024/beach.jpg":        originalJPEG,
			"2024/beach (1).jpg":    originalJPEG,
			"backup/IMG_0001.JPG":   originalJPEG,
			"edits/beach-q40.jpg":   encodeJPEG(t, original, 40),
			"edits/beach-small.jpg": encodeJPEG(t, beach.render(97, 73), 70),
			"edits/beach-rot.jpg":   rotatedWithTag(t, original),
			"edits/beach-trim.jpg":  encodeJPEG(t, crop(original, image.Rect(8, 6, 312, 234)), 90),
			"edits/beach-crop.jpg":  encodeJPEG(t, crop(original, image.Rect(32, 24, 288, 216)), 90),
			// A quarter cropped off is a different picture to a dHash.
			"edits/beach-half.jpg": encodeJPEG(t, crop(original, image.Rect(80, 0, 320, 240)), 90),
			// The same pixels in other bytes: identical to look at, but not
			// the same file, so never exact.
			"harbor.jpg": harborJPEG,
			"harbor.png": encodePNG(t, harbor),
			// Hard negatives: the beach's colors, sun and hills, other skies.
			"sunsets/a.jpg": encodeJPEG(t, safetyScene{clouds: 2, sunX: 0.7, subjectX: -1}.render(320, 240), 90),
			"sunsets/b.jpg": encodeJPEG(t, safetyScene{clouds: 3, sunX: 0.3, subjectX: -1}.render(320, 240), 90),
			"sunsets/c.jpg": encodeJPEG(t, safetyScene{clouds: 4, sunX: 0.3, subjectX: -1}.render(320, 240), 90),
			"sunsets/d.jpg": encodeJPEG(t, safetyScene{clouds: 5, sunX: 0.3, hillPhase: 2, subjectX: -1}.render(320, 240), 90),
		},
		usbSerial: {
			// The harbor's exact copy on the second device.
			"camera/harbor.jpg": harborJPEG,
		},
	})

	got := lib.groups(t)
	// Near groups come most similar first: the harbor's copies share pixels.
	want := []string{
		"near: |harbor.jpg |harbor.png " + usbSerial + "|camera/harbor.jpg",
		"near: |2024/beach (1).jpg |2024/beach.jpg |backup/IMG_0001.JPG |edits/beach-crop.jpg" +
			" |edits/beach-q40.jpg |edits/beach-rot.jpg |edits/beach-small.jpg |edits/beach-trim.jpg",
	}
	if gotStrings := groupStrings(got); !reflect.DeepEqual(gotStrings, want) {
		for _, p := range []string{
			"|edits/beach-q40.jpg", "|edits/beach-small.jpg", "|edits/beach-rot.jpg",
			"|edits/beach-trim.jpg", "|edits/beach-crop.jpg", "|edits/beach-half.jpg",
			"|sunsets/a.jpg", "|sunsets/b.jpg", "|sunsets/c.jpg", "|sunsets/d.jpg",
		} {
			t.Logf("distance beach -> %s = %d", p, lib.distance(t, "|2024/beach.jpg", p))
		}
		t.Fatalf("groups =\n%q\nwant\n%q", gotStrings, want)
	}
	// A group is clustered around a seed, so every member is within the
	// threshold of the original, though two edits may be further apart.
	for _, p := range got[1].Photos {
		if d := lib.distance(t, "|2024/beach.jpg", "|"+p.RelPath); d > photoutil.ParseDuplicateThreshold("") {
			t.Errorf("%s is %d from the original, over the threshold", p.RelPath, d)
		}
	}

	// The near groups above hold the exact copies, since each photo is in one
	// group (#1666). Without the transformed copies the exact copies stand
	// alone as exact groups, and a file with other bytes still never joins.
	t.Run("exact copies alone", func(t *testing.T) {
		lib := newSafetyLibrary(t, map[string]map[string][]byte{
			"": {
				"2024/beach.jpg":      originalJPEG,
				"2024/beach (1).jpg":  originalJPEG,
				"backup/IMG_0001.JPG": originalJPEG,
				"harbor.jpg":          harborJPEG,
			},
			usbSerial: {"camera/harbor.jpg": harborJPEG, "camera/beach.jpg": originalJPEG},
		})
		got := groupStrings(lib.groups(t))
		want := []string{
			"exact: |2024/beach (1).jpg |2024/beach.jpg |backup/IMG_0001.JPG " + usbSerial + "|camera/beach.jpg",
			"exact: |harbor.jpg " + usbSerial + "|camera/harbor.jpg",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("groups = %q, want %q", got, want)
		}
	})

	t.Run("same pixels in other bytes are not exact", func(t *testing.T) {
		lib := newSafetyLibrary(t, map[string]map[string][]byte{
			"": {"harbor.jpg": harborJPEG, "harbor.png": encodePNG(t, harbor)},
		})
		groups := lib.groups(t)
		if got, want := groupStrings(groups), []string{"near: |harbor.jpg |harbor.png"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("groups = %q, want %q", got, want)
		}
	})

	t.Run("hard negatives never group", func(t *testing.T) {
		files := map[string][]byte{}
		for i, s := range []safetyScene{
			{clouds: 1, sunX: 0.3, subjectX: -1},
			{clouds: 2, sunX: 0.7, subjectX: -1},
			{clouds: 3, sunX: 0.3, subjectX: -1},
			{clouds: 4, sunX: 0.3, subjectX: -1},
			{clouds: 5, sunX: 0.3, hillPhase: 2, subjectX: -1},
			{clouds: 6, sunX: 0.5, hillPhase: 4, subjectX: -1},
		} {
			files[fmt.Sprintf("sunsets/%d.jpg", i)] = encodeJPEG(t, s.render(320, 240), 90)
		}
		files["edits/beach-half.jpg"] = encodeJPEG(t, crop(original, image.Rect(80, 0, 320, 240)), 90)
		lib := newSafetyLibrary(t, map[string]map[string][]byte{"": files})
		if got := groupStrings(lib.groups(t)); len(got) != 0 {
			t.Fatalf("false groups = %q, want none", got)
		}
	})
}

// TestDuplicatesSafety_ThresholdBoundary pins the near-duplicate threshold of
// 10 to the bit: photos whose stored dHashes are 10 apart group, 11 apart do
// not. The images are drawn to hash to known values and stored losslessly,
// so the distance is exact after a real decode.
func TestDuplicatesSafety_ThresholdBoundary(t *testing.T) {
	const at, over = 0x0f0f0f0f0f0f0f0f, 0xf0f0f0f0f0f0f0f0
	files := map[string][]byte{
		"at/a.png":   encodePNG(t, hashImage(at)),
		"at/b.png":   encodePNG(t, hashImage(flipBits(at, 10))),
		"over/a.png": encodePNG(t, hashImage(over)),
		"over/b.png": encodePNG(t, hashImage(flipBits(over, 11))),
	}
	lib := newSafetyLibrary(t, map[string]map[string][]byte{"": files})

	hashes := lib.storedHashes(t)
	for p, want := range map[string]uint64{
		"at/a.png": at, "at/b.png": flipBits(at, 10), "over/a.png": over, "over/b.png": flipBits(over, 11),
	} {
		if got := hashes["|"+p]; got != fmt.Sprintf("%016x", want) {
			t.Fatalf("%s stored dHash %s, want %016x", p, got, want)
		}
	}
	if d := lib.distance(t, "|at/a.png", "|at/b.png"); d != 10 {
		t.Fatalf("at pair distance = %d, want 10", d)
	}
	if d := lib.distance(t, "|over/a.png", "|over/b.png"); d != 11 {
		t.Fatalf("over pair distance = %d, want 11", d)
	}

	groups := lib.groups(t)
	if got, want := groupStrings(groups), []string{"near: |at/a.png |at/b.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("groups = %q, want %q", got, want)
	}
	if groups[0].MaxDistance != 10 {
		t.Errorf("MaxDistance = %d, want 10", groups[0].MaxDistance)
	}
}

// TestDuplicatesSafety_BurstAndLivePhoto pins what duplicate detection does with the two
// kinds of shot a phone takes several files of. A burst's frames are near
// duplicates and are grouped, so the user can keep the best one. A Live
// Photo's video is not a photo: it is never hashed and never offered as a
// copy, of its own still or anything else, even when the still has a copy
// and the video does too.
func TestDuplicatesSafety_BurstAndLivePhoto(t *testing.T) {
	burst := map[string][]byte{}
	for i := range 4 {
		frame := safetyScene{clouds: 11, sunX: 0.3, subjectX: 0.6 + 0.03*float64(i)}
		burst[fmt.Sprintf("burst/IMG_%04d.JPG", i+1)] = encodeJPEG(t, frame.render(320, 240), 90)
	}
	live := encodeJPEG(t, safetyScene{clouds: 21, sunX: 0.5, hillPhase: 3, subjectX: -1}.render(320, 240), 90)
	video := []byte("live photo video")
	other := encodeJPEG(t, safetyScene{clouds: 31, sunX: 0.2, hillPhase: 5, subjectX: -1}.render(320, 240), 90)

	files := map[string][]byte{
		"live/IMG_0100.JPG":   live,
		"live/IMG_0100.MOV":   video,
		"backup/IMG_0100.JPG": live,
		"backup/IMG_0100.MOV": video,
		// A Live Photo with no copy is in no group at all.
		"live/IMG_0200.JPG": other,
		"live/IMG_0200.MOV": video,
	}
	for p, data := range burst {
		files[p] = data
	}
	lib := newSafetyLibrary(t, map[string]map[string][]byte{"": files})

	got := groupStrings(lib.groups(t))
	want := []string{
		"exact: |backup/IMG_0100.JPG |live/IMG_0100.JPG",
		"near: |burst/IMG_0001.JPG |burst/IMG_0002.JPG |burst/IMG_0003.JPG |burst/IMG_0004.JPG",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("groups = %q, want %q", got, want)
	}
	for key := range lib.storedHashes(t) {
		if strings.HasSuffix(key, ".MOV") {
			t.Errorf("a Live Photo video was hashed: %s", key)
		}
	}

	page, err := photoutil.ListPhotos(photoutil.ListPhotosParams{
		Ctx: lib.ctx, Registry: lib.registry, Access: systemAccess(t), Limit: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range page.Photos {
		if p.RelPath == "live/IMG_0100.JPG" && !p.HasLiveVideo {
			t.Error("the grouped still is not marked as a Live Photo")
		}
	}
}

// TestDuplicatesSafety_DeleteKeepsTheKept deletes the copies marked in a
// group through the real delete path and requires the kept file to be the
// one left: still on its device, byte for byte, decodable, in the album
// that held both, and out of the duplicates list. A multi-format group (the
// same shot as PNG and JPEG) loses only the format chosen, either way round.
func TestDuplicatesSafety_DeleteKeepsTheKept(t *testing.T) {
	shot := safetyScene{clouds: 41, sunX: 0.4, hillPhase: 1, subjectX: -1}.render(320, 240)
	shotJPEG := encodeJPEG(t, shot, 90)
	shotPNG := encodePNG(t, shot)

	t.Run("exact copies across devices", func(t *testing.T) {
		lib := newSafetyLibrary(t, map[string]map[string][]byte{
			"":        {"2024/shot.jpg": shotJPEG, "2024/shot (1).jpg": shotJPEG},
			usbSerial: {"camera/shot.jpg": shotJPEG},
		})
		q := lib.database.Queries
		owner, err := q.CreateUser(lib.ctx, db.CreateUserParams{Username: "owner", PasswordHash: "h", RecoveryPhraseHash: "r"})
		if err != nil {
			t.Fatal(err)
		}
		album, err := q.CreateAlbum(lib.ctx, db.CreateAlbumParams{Name: "Trip", UserID: owner.ID})
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []struct{ serial, path string }{{"", "2024/shot.jpg"}, {"", "2024/shot (1).jpg"}, {usbSerial, "camera/shot.jpg"}} {
			if _, err := q.AddPhotoToAlbum(lib.ctx, db.AddPhotoToAlbumParams{AlbumID: album.ID, DeviceSerial: p.serial, RelPath: p.path}); err != nil {
				t.Fatal(err)
			}
		}
		if got := groupStrings(lib.groups(t)); len(got) != 1 || !strings.HasPrefix(got[0], "exact:") {
			t.Fatalf("groups = %q, want one exact group", got)
		}
		_, before := lib.readable(t, "", "2024/shot.jpg")

		// Keep the internal original; delete the internal copy and the USB copy.
		if err := lib.deleteCopies(t, "", "2024/shot (1).jpg"); err != nil {
			t.Fatal(err)
		}
		if err := lib.deleteCopies(t, usbSerial, "camera/shot.jpg"); err != nil {
			t.Fatal(err)
		}

		ok, after := lib.readable(t, "", "2024/shot.jpg")
		if !ok || after != before {
			t.Fatalf("kept file readable = %v, unchanged = %v", ok, after == before)
		}
		for _, gone := range []struct{ serial, path string }{{"", "2024/shot (1).jpg"}, {usbSerial, "camera/shot.jpg"}} {
			if ok, _ := lib.readable(t, gone.serial, gone.path); ok {
				t.Errorf("%s|%s is still on its device", gone.serial, gone.path)
			}
		}
		if got := groupStrings(lib.groups(t)); len(got) != 0 {
			t.Errorf("groups after delete = %q, want none", got)
		}
		wantAlbum := []string{"|2024/shot.jpg"}
		var gotAlbum []string
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
			gotAlbum = albumItemKeys(t, q, album.ID)
			if reflect.DeepEqual(gotAlbum, wantAlbum) {
				break
			}
		}
		if !reflect.DeepEqual(gotAlbum, wantAlbum) {
			t.Errorf("album = %q, want only the kept file %q", gotAlbum, wantAlbum)
		}
	})

	for _, tc := range []struct{ keep, drop string }{
		{keep: "shot.png", drop: "shot.jpg"},
		{keep: "shot.jpg", drop: "shot.png"},
	} {
		t.Run("multi-format keeps "+path.Ext(tc.keep), func(t *testing.T) {
			lib := newSafetyLibrary(t, map[string]map[string][]byte{
				"": {"shot.jpg": shotJPEG, "shot.png": shotPNG},
			})
			if got, want := groupStrings(lib.groups(t)), []string{"near: |shot.jpg |shot.png"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("groups = %q, want %q", got, want)
			}
			_, before := lib.readable(t, "", tc.keep)
			if err := lib.deleteCopies(t, "", tc.drop); err != nil {
				t.Fatal(err)
			}
			if ok, after := lib.readable(t, "", tc.keep); !ok || after != before {
				t.Fatalf("kept %s readable = %v, unchanged = %v", tc.keep, ok, after == before)
			}
			if ok, _ := lib.readable(t, "", tc.drop); ok {
				t.Errorf("%s is still on the device", tc.drop)
			}
			if got := groupStrings(lib.groups(t)); len(got) != 0 {
				t.Errorf("groups after delete = %q, want none", got)
			}
		})
	}
}

// TestDuplicatesSafety_FailedDeleteKeepsTheFile: a copy the device refuses to
// trash is reported as an error and stays exactly where it was, still hashed
// and still offered in its group, so the user can try again.
func TestDuplicatesSafety_FailedDeleteKeepsTheFile(t *testing.T) {
	shotJPEG := encodeJPEG(t, safetyScene{clouds: 51, sunX: 0.6, subjectX: -1}.render(320, 240), 90)
	lib := newSafetyLibrary(t, map[string]map[string][]byte{
		"": {"shot.jpg": shotJPEG, "shot copy.jpg": shotJPEG},
	})
	lib.devices[""].fail["shot copy.jpg"] = true
	_, before := lib.readable(t, "", "shot copy.jpg")

	if err := lib.deleteCopies(t, "", "shot copy.jpg"); err == nil {
		t.Fatal("DeleteFiles reported success for a copy the device refused to trash")
	}
	for _, p := range []string{"shot.jpg", "shot copy.jpg"} {
		if ok, after := lib.readable(t, "", p); !ok || after != before {
			t.Errorf("%s readable = %v, unchanged = %v", p, ok, after == before)
		}
	}
	if got, want := groupStrings(lib.groups(t)), []string{"exact: |shot copy.jpg |shot.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("groups after the failed delete = %q, want %q", got, want)
	}
}

// TestDuplicatesSafety_MissingHashes: a photo moved or deleted outside Quark,
// or on a drive since unplugged, still has stored hashes. Listing duplicates
// does not fail on it and never offers it: a group it leaves with one photo
// disappears, a larger group keeps the rest, and the missing photo's hashes
// are dropped.
func TestDuplicatesSafety_MissingHashes(t *testing.T) {
	a := encodeJPEG(t, safetyScene{clouds: 61, sunX: 0.3, subjectX: -1}.render(320, 240), 90)
	b := encodeJPEG(t, safetyScene{clouds: 62, sunX: 0.7, hillPhase: 2, subjectX: -1}.render(320, 240), 90)
	c := encodeJPEG(t, safetyScene{clouds: 63, sunX: 0.5, hillPhase: 4, subjectX: -1}.render(320, 240), 90)
	lib := newSafetyLibrary(t, map[string]map[string][]byte{
		"":        {"a/1.jpg": a, "a/2.jpg": a, "a/3.jpg": a, "b/1.jpg": b, "b/2.jpg": b, "c/1.jpg": c},
		usbSerial: {"c/1.jpg": c},
	})
	if got := groupStrings(lib.groups(t)); len(got) != 3 {
		t.Fatalf("groups before = %q, want three", got)
	}

	for _, p := range []string{"a/3.jpg", "b/2.jpg"} {
		if err := lib.devices[""].Delete(lib.ctx, p, vfs.DeleteOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	lib.registry.Unregister(vfs.FilesNamespace(usbSerial))

	got := groupStrings(lib.groups(t))
	if want := []string{"exact: |a/1.jpg |a/2.jpg"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("groups = %q, want %q", got, want)
	}
	hashes := lib.storedHashes(t)
	for _, key := range []string{"|a/3.jpg", "|b/2.jpg"} {
		if _, ok := hashes[key]; ok {
			t.Errorf("hashes of missing %s are still stored", key)
		}
	}
	if _, ok := hashes["|b/1.jpg"]; !ok {
		t.Error("the photo left alone in its group lost its hashes")
	}
}

// albumItemKeys lists an album's photos as "serial|path", sorted.
func albumItemKeys(t *testing.T, q db.Querier, albumID int64) []string {
	t.Helper()
	rows, err := q.ListAlbumItems(context.Background(), albumID)
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for _, r := range rows {
		keys = append(keys, r.DeviceSerial+"|"+r.RelPath)
	}
	sort.Strings(keys)
	return keys
}
