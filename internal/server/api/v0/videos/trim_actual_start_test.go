package v0_videos_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/videoutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// sparseGOPFixture is four seconds at 24 fps with keyframes at 0 and 3
// seconds only, so a start between them carries the longest lead-in a trim
// can. videoutil's trim_actual_start_test.go derives the start each cut of it
// is shown from out of its keyframe table; these tests pin that the API
// reports that same start, so neither hop can lose it (#3430).
const sparseGOPFixture = "../../../../../pkg/util/videoutil/testdata/h264-gop72.mp4"

// sparseCuts are the trims of the sparse-GOP fixture: the issue's live case
// between the keyframes, one on the second keyframe, and one a millisecond
// into the last frame, the latest start a trim accepts.
var sparseCuts = []struct {
	name           string
	startMs, endMs int64
}{
	{"between the keyframes", 1800, 3500},
	{"on the second keyframe", 3000, 3500},
	{"inside the last frame", 3959, 4000},
}

// trimmed is a 200 trim response.
type trimmed struct {
	RelPath       string `json:"relPath"`
	ActualStartMs int64  `json:"actualStartMs"`
}

// trimOK posts a trim of rel and reads the 200 it must answer.
func (h harness) trimOK(t *testing.T, rel string, startMs, endMs int64) trimmed {
	t.Helper()
	w := h.trim(fmt.Sprintf(`{"relPath":%q,"serial":%q,"startMs":%d,"endMs":%d}`, rel, testSerial, startMs, endMs))
	if w.Code != http.StatusOK {
		t.Fatalf("trim returned %d, want 200: %s", w.Code, w.Body.String())
	}
	var body trimmed
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

// libraryStart is the start videoutil.TrimFile reports for the same cut of
// the same bytes, in a namespace of its own.
func libraryStart(t *testing.T, startMs, endMs int64) int64 {
	t.Helper()
	fsys := vfs.NewMemVFS("files")
	f, err := os.Open(sparseGOPFixture)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := fsys.Write(context.Background(), "clip.mp4", f, vfs.WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	result, err := videoutil.TrimFile(context.Background(), videoutil.TrimFileParams{
		FS: fsys, Path: "clip.mp4",
		Start: time.Duration(startMs) * time.Millisecond, End: time.Duration(endMs) * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Start.Milliseconds()
}

// TestTrimReportsTheSparseGOPStart trims the sparse-GOP fixture through the
// API: actualStartMs is the start TrimFile works out from the keyframes, and
// the clip lasts from there to the requested end, give or take a frame.
func TestTrimReportsTheSparseGOPStart(t *testing.T) {
	const frame = time.Second / 24
	for _, c := range sparseCuts {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t)
			copyFixture(t, sparseGOPFixture, filepath.Join(h.filesDir, "clip.mp4"))
			body := h.trimOK(t, "clip.mp4", c.startMs, c.endMs)
			if want := libraryStart(t, c.startMs, c.endMs); body.RelPath != "clip_trimmed.mp4" || body.ActualStartMs != want {
				t.Fatalf("trim = %+v, want clip_trimmed.mp4 shown from %dms", body, want)
			}
			info, err := videoutil.Probe(context.Background(), filepath.Join(h.filesDir, body.RelPath))
			if err != nil {
				t.Fatal(err)
			}
			want := time.Duration(c.endMs-body.ActualStartMs) * time.Millisecond
			if d := info.Duration - want; d < -frame || d > frame {
				t.Errorf("clip lasts %v, want %v from actualStartMs to endMs, give or take a frame", info.Duration, want)
			}
		})
	}
}

// TestTrimNumberedClipsReportTheirOwnStart trims one video from three
// starts: each clip takes the next free name and reports its own start,
// never the one before it.
func TestTrimNumberedClipsReportTheirOwnStart(t *testing.T) {
	h := newHarness(t)
	copyFixture(t, sparseGOPFixture, filepath.Join(h.filesDir, "clip.mp4"))
	names := []string{"clip_trimmed.mp4", "clip_trimmed_(1).mp4", "clip_trimmed_(2).mp4"}
	for i, c := range sparseCuts {
		body := h.trimOK(t, "clip.mp4", c.startMs, c.endMs)
		if want := libraryStart(t, c.startMs, c.endMs); body.RelPath != names[i] || body.ActualStartMs != want {
			t.Errorf("trim %d (%s) = %+v, want %s shown from %dms", i, c.name, body, names[i], want)
		}
	}
}

// cancelingVFS is a namespace whose Write cancels the request that asked for
// it once the clip's first bytes have arrived, the way a client that gives up
// mid-trim does, and then lets the real Write carry on. It counts the bytes
// the trim managed to send.
type cancelingVFS struct {
	vfs.VFS
	cancel context.CancelFunc
	sent   atomic.Int64
}

func (c *cancelingVFS) Write(ctx context.Context, p string, r io.Reader, opts vfs.WriteOptions) error {
	r = &countingReader{r: r, n: &c.sent}
	first := make([]byte, 1)
	if _, err := io.ReadFull(r, first); err != nil {
		return err
	}
	c.cancel()
	return c.VFS.Write(ctx, p, io.MultiReader(strings.NewReader(string(first)), r), opts)
}

type countingReader struct {
	r io.Reader
	n *atomic.Int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

// TestTrimStopsWhenTheClientGoesAway cancels a trim's request while the clip
// is being written: the trim stops short of the whole clip, nothing is left
// under any _trimmed name or anywhere else in the folder, and no upload event
// tells a file browser to list a clip that is not there.
func TestTrimStopsWhenTheClientGoesAway(t *testing.T) {
	h := newHarness(t)
	copyFixture(t, sparseGOPFixture, filepath.Join(h.filesDir, "clip.mp4"))
	whole := h.trimOK(t, "clip.mp4", 1800, 3500)
	stat, err := os.Stat(filepath.Join(h.filesDir, whole.RelPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(h.filesDir, whole.RelPath)); err != nil {
		t.Fatal(err)
	}
	<-h.events

	// Put the canceling namespace where the disk's is.
	id := vfs.FilesNamespace(testSerial)
	i := slices.IndexFunc(h.registry.List(""), func(ns vfs.Namespace) bool { return ns.ID == id })
	if i < 0 {
		t.Fatalf("no namespace %s registered", id)
	}
	ns := h.registry.List("")[i]
	disk, _ := h.registry.Get(id)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	canceling := &cancelingVFS{VFS: disk, cancel: cancel}
	h.registry.Unregister(id)
	if err := h.registry.Register(ns, canceling); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v0/videos/trim",
		strings.NewReader(fmt.Sprintf(`{"relPath":"clip.mp4","serial":%q,"startMs":1800,"endMs":3500}`, testSerial)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.engine.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("a canceled trim answered 200: %s", w.Body.String())
	}
	if sent := canceling.sent.Load(); sent >= stat.Size() {
		t.Errorf("the trim sent %d bytes after its client left, the whole %d-byte clip", sent, stat.Size())
	}
	left, err := os.ReadDir(h.filesDir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range left {
		got = append(got, e.Name())
	}
	if !slices.Equal(got, []string{"clip.mp4"}) {
		t.Errorf("the folder holds %v after a canceled trim, want only clip.mp4", got)
	}
	select {
	case e := <-h.events:
		if e.Kind == eventbus.EventUpload {
			t.Errorf("a canceled trim published %+v", e)
		}
	default:
	}
}
