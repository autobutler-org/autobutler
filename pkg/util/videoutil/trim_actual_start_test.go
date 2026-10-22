package videoutil

// cspell:words stss

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"testing"
	"time"
)

// sparseGOPFixture is h264-gop72.mp4 from sprocket's corpus: four seconds of
// the same 128x72, 24 fps content with keyframes at 0 and 3 seconds only, the
// sparse GOP a phone or a screen recorder writes. A start between them has
// only the keyframe at zero to copy from, so the clip carries the longest
// lead-in a trim of it can, and actualStartMs is where that shows (#3430).
const sparseGOPFixture = "testdata/h264-gop72.mp4"

// keyframeTable is the presentation time of every keyframe in the MP4 at p,
// read from its sync sample box and its frame rate, and the time its video
// ends: its last frame's time plus that frame's duration. Every expected start below is derived from it rather than
// written down, so a regenerated fixture moves the expectations with it.
func keyframeTable(t *testing.T, p string) (keyframes []time.Duration, videoEnd time.Duration) {
	t.Helper()
	info, err := Probe(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture is a few dozen kilobytes, so reading it whole is fine.
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	// stss: size, type, version and flags, entry count, then one 1-based
	// sample number per keyframe. The fixture has one video track, so it has
	// one stss, and a constant frame rate with no B-frame before a keyframe,
	// so sample n is shown at (n-1)/fps.
	at := bytes.Index(data, []byte("stss"))
	if at < 4 {
		t.Fatalf("%s has no sync sample box", p)
	}
	count := binary.BigEndian.Uint32(data[at+8:])
	for i := range count {
		n := binary.BigEndian.Uint32(data[at+12+4*int(i):])
		keyframes = append(keyframes, frames(int(n-1), info.Framerate))
	}
	// The fixture's audio ends with its video, so the probed duration is the
	// video's.
	return keyframes, info.Duration
}

// frames is how long n frames last at fps.
func frames(n int, fps float64) time.Duration {
	return time.Duration(float64(n) * float64(time.Second) / fps)
}

// shownFrom is where sprocket's documented Trim shows a clip from, worked out
// from the source's keyframes: the requested start, its lead-in hidden,
// unless there is nothing to hide or nothing to hide it behind. A start on a
// keyframe, or one into a TS output, is shown from the keyframe at or before
// it; a start at or past the end of the video takes the last keyframe. A start
// inside the last frame still shows that frame, lead-in hidden.
func shownFrom(keyframes []time.Duration, videoEnd, start time.Duration, ts bool) time.Duration {
	if start >= videoEnd {
		return keyframes[len(keyframes)-1]
	}
	snapped := keyframes[0]
	for _, k := range keyframes {
		if k <= start {
			snapped = k
		}
	}
	if snapped == start || ts {
		return snapped
	}
	return start
}

// within reports whether got is want give or take one frame at fps.
func within(got, want time.Duration, fps float64) bool {
	d := got - want
	return d >= -frames(1, fps) && d <= frames(1, fps)
}

// sparseCut is one trim of the sparse-GOP fixture and the start the keyframe
// table says it is shown from.
type sparseCut struct {
	name       string
	start, end time.Duration
}

// sparseCuts are the trims every sparse-GOP test runs: the issue's live case
// between the keyframes, one on the second keyframe, and one a millisecond
// into the last frame, the latest start TrimFile accepts.
func sparseCuts(videoEnd time.Duration) []sparseCut {
	lastFrame := videoEnd - frames(1, 24)
	return []sparseCut{
		{"between the keyframes", 1800 * time.Millisecond, 3500 * time.Millisecond},
		{"on the second keyframe", 3000 * time.Millisecond, 3500 * time.Millisecond},
		{"inside the last frame", lastFrame + time.Millisecond, videoEnd},
	}
}

// TestTrimFileReportsTheSparseGOPStart trims the sparse-GOP fixture through
// TrimFile and checks each clip reports the start its keyframe table gives,
// and lasts from there to the end it was asked for, within a frame.
func TestTrimFileReportsTheSparseGOPStart(t *testing.T) {
	keyframes, videoEnd := keyframeTable(t, sparseGOPFixture)
	if len(keyframes) != 2 || keyframes[0] != 0 || keyframes[1] <= 1800*time.Millisecond {
		t.Fatalf("keyframes = %v, want the sparse GOP: one at zero and the next past 1.8s", keyframes)
	}
	const fps = 24
	for _, c := range sparseCuts(videoEnd) {
		t.Run(c.name, func(t *testing.T) {
			fsys := memWith(t, sparseGOPFixture, "clip.mp4")
			result, err := TrimFile(context.Background(), TrimFileParams{FS: fsys, Path: "clip.mp4", Start: c.start, End: c.end})
			if err != nil {
				t.Fatal(err)
			}
			want := shownFrom(keyframes, videoEnd, c.start, false)
			if result.Start != want {
				t.Fatalf("Start = %v, want %v from keyframes %v", result.Start, want, keyframes)
			}
			clip, err := OpenSource(context.Background(), OpenSourceParams{FS: fsys, Path: result.Path})
			if err != nil {
				t.Fatal(err)
			}
			defer clip.Close()
			info, err := ProbeSource(context.Background(), clip)
			if err != nil {
				t.Fatal(err)
			}
			if !within(info.Duration, c.end-want, fps) {
				t.Errorf("clip lasts %v, want %v from %v to %v, give or take a frame", info.Duration, c.end-want, want, c.end)
			}
		})
	}
}

// TestTrimReportsTheKeyframeWhereItCannotHide reaches the edges TrimFile
// never asks Trim for, where the clip starts on a keyframe instead of the
// request. A TS output shows its lead-in: TrimFile never writes one, since a
// TS source cannot be cut, but Trim takes its format from the name, so an MP4
// named .ts goes out as TS. A start at or past the end of the video, which
// TrimFile refuses, takes the last keyframe.
func TestTrimReportsTheKeyframeWhereItCannotHide(t *testing.T) {
	keyframes, videoEnd := keyframeTable(t, sparseGOPFixture)
	cases := []struct {
		name, as   string
		start, end time.Duration
	}{
		{"a TS output", "clip.ts", 1800 * time.Millisecond, 3500 * time.Millisecond},
		{"a start at the end", "clip.mp4", videoEnd, videoEnd + time.Second},
		{"a start past the end", "clip.mp4", videoEnd + time.Second, videoEnd + 2*time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := openFixture(t, sparseGOPFixture)
			src.Name = c.as
			result, err := Trim(context.Background(), TrimParams{Source: src, Output: io.Discard, Start: c.start, End: c.end})
			if err != nil {
				t.Fatal(err)
			}
			want := shownFrom(keyframes, videoEnd, c.start, TrimFormat(c.as) == "ts")
			if result.Start != want || want == c.start {
				t.Fatalf("Start = %v, want the keyframe %v rather than the requested %v", result.Start, want, c.start)
			}
		})
	}
}

// TestTrimFileNumberedClipsReportTheirOwnStart trims the same video three
// times, each from a different start: every clip takes the next free name
// and reports where it starts, never the start of the clip before it.
func TestTrimFileNumberedClipsReportTheirOwnStart(t *testing.T) {
	keyframes, videoEnd := keyframeTable(t, sparseGOPFixture)
	fsys := memWith(t, sparseGOPFixture, "clip.mp4")
	names := []string{"clip_trimmed.mp4", "clip_trimmed_(1).mp4", "clip_trimmed_(2).mp4"}
	for i, c := range sparseCuts(videoEnd) {
		result, err := TrimFile(context.Background(), TrimFileParams{FS: fsys, Path: "clip.mp4", Start: c.start, End: c.end})
		if err != nil {
			t.Fatal(err)
		}
		want := shownFrom(keyframes, videoEnd, c.start, false)
		if result.Path != names[i] || result.Start != want {
			t.Errorf("trim %d (%s) = %+v, want %s shown from %v", i, c.name, result, names[i], want)
		}
	}
}
