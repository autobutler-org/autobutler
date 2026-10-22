// Package odputil reads an OpenDocument presentation (.odp) and writes it as
// the .qslide JSON Quark's slides editor and viewer already understand — the
// same file pptxutil's import writes for a .pptx, so one viewer and one
// editor serve both formats.
//
// The package opens the file through officezip: the caller supplies an
// [io.ReaderAt] and its size, and every part read is bounded. styles.xml —
// the named styles, the page size and the master pages — is read first and
// held, since every slide names its styles and master by name. content.xml is
// then decoded as a token stream: its automatic styles are held, and each
// draw:page is read, converted and written to Out before the next is read, so
// one slide is held at a time.
//
// What carries across: the slide size, as the editor's 1920×1080 or 1024×768
// preset when the page is 16:9 or 4:3; text frames with their paragraphs and
// runs (bold, italic, underline, strikethrough, size, font, color), alignment,
// line spacing, bullets and numbering, vertical anchor and auto-fit; pictures
// with their alt text; rectangles, rounded rectangles, ellipses, triangles,
// diamonds, right arrows, stars and polygons with their fill, outline,
// opacity and shadow; lines and connectors, with arrowheads; groups, nested;
// tables, with their column widths, row heights, cell text, fills, borders
// and merged cells; rotation; stacking order; slide backgrounds, a color or a
// picture, from the slide or its master; the master page's own shapes,
// drawn behind every slide that shows them; fade, push and wipe transitions;
// and speaker notes.
//
// Styles are resolved as LibreOffice resolves them: a run takes what its
// span's style sets, then its paragraph's, then its frame's, each through its
// parent styles to the family's default.
//
// What the editor has no model for — charts and embedded objects, video and
// audio, animations, freeform paths, gradient and hatch fills, picture fills
// on shapes, skewed shapes — is left out or approximated and named in the
// slide's warnings, as pptxutil does. Linked pictures are never fetched.
// Macros and scripts are never opened. An encrypted content.xml is
// [ErrEncrypted].
package odputil

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// Import limits. An .odp is user-supplied, so each is checked against what
// is actually read.
const (
	// MaxSlides is the number of slides one import writes.
	MaxSlides = 5_000
	// MaxElements bounds the elements of the whole deck, grouped ones and
	// table cells included.
	MaxElements = 200_000
	// MaxGroupDepth bounds how deeply groups nest.
	MaxGroupDepth = 32
	// MaxImportBytes bounds what a package unpacks to, every part together.
	MaxImportBytes = 2 << 30 // 2 GiB
	// MaxPartBytes bounds one part unpacked: content.xml, styles.xml, a
	// picture. A larger picture is left out with a warning.
	MaxPartBytes = officezip.DefaultMaxPartBytes
	// MaxMediaBytes bounds the pictures of one import together; past it,
	// the rest are left out with a warning.
	MaxMediaBytes = 1 << 30 // 1 GiB
)

var (
	// ErrMalformed is a file that is not a well-formed OpenDocument
	// presentation: not a zip, a zip with no presentation, or a part that
	// does not parse. It is [officezip.ErrMalformed], so either name
	// matches. The caller's file is at fault, so callers answer 400 for it.
	ErrMalformed = officezip.ErrMalformed
	// ErrTooLarge is a presentation past a limit: the package's own, or one
	// of the import limits above. It is [officezip.ErrTooLarge], so either
	// name matches. Also the caller's file, and also a 400.
	ErrTooLarge = officezip.ErrTooLarge
	// ErrEncrypted is a presentation whose content.xml is password-protected.
	// The bytes are not decrypted and not parsed.
	ErrEncrypted = errors.New("odputil: the presentation is encrypted")
)

// StoreMediaFunc stores a picture an import found, under a name like
// image1.png whose extension matches its content, and returns the reference
// the .qslide names it by. Part is the picture's entry in the package, such
// as Pictures/10000000000001.png, for a caller that refers to it in place
// rather than storing it. It reads r to EOF, even when it keeps nothing: the
// size limits are checked as r is read, so a picture read only in part is
// never found past them. r fails with [ErrTooLarge] past [MaxPartBytes], and
// a picture that fails that way is skipped with a warning. Any other error
// fails the import.
//
// It has the shape of pptxutil.StoreMediaFunc, so one function serves both
// importers.
type StoreMediaFunc func(part, name string, r io.Reader) (string, error)

// ImportOdpParams is one .odp on its way to becoming a .qslide.
type ImportOdpParams struct {
	// Ctx ends the import when it is done. Nil never does. A canceled
	// context returns that error and stops reading.
	Ctx context.Context
	// Source is the package, read in place: a zip's directory is at its end.
	Source io.ReaderAt
	// Size is Source's length in bytes, from the source's own Stat.
	Size int64
	// Out receives the .qslide JSON, a slide at a time.
	Out io.Writer
	// StoreMedia stores the pictures the slides show. Each is stored once,
	// however many slides show it. Nil skips every picture, with a warning.
	StoreMedia StoreMediaFunc
	// Title is the presentation's title when the package names none.
	Title string
}

// ImportOdpResult reports what the imported presentation came to.
type ImportOdpResult struct {
	// Slides is the number of slides written.
	Slides int
	// Pictures is the number of distinct pictures stored.
	Pictures int
	// Warnings lists what was left out or approximated, slide by slide.
	Warnings []ImportWarning
}

// ImportWarning is one thing an import left out or approximated.
type ImportWarning struct {
	// Slide is the slide's number in show order, from 1; 0 is the deck.
	Slide int
	// Message says what, in a sentence a user can read.
	Message string
}

// ImportOdp reads the .odp in params.Source and writes the equivalent
// .qslide to params.Out, one slide per draw:page in show order.
//
// What the editor cannot show is skipped or approximated and reported in
// the result's warnings — a chart, a video, an animation, a shape kind the
// editor does not have drawn as a rectangle — and is never a failure. Errors
// wrap [ErrMalformed] when the source is not an OpenDocument presentation,
// [ErrTooLarge] past the import limits and [ErrEncrypted] when content.xml
// is password-protected. A canceled params.Ctx returns that context's error.
// Anything else is a read, write or StoreMedia failure. Out may hold part of
// a .qslide when an error is returned.
func ImportOdp(params ImportOdpParams) (ImportOdpResult, error) {
	if params.Source == nil || params.Out == nil {
		return ImportOdpResult{}, errors.New("odputil: Source and Out are required")
	}
	if params.Size <= 0 {
		return ImportOdpResult{}, fmt.Errorf("%w: empty file", ErrMalformed)
	}
	if err := canceled(params.Ctx); err != nil {
		return ImportOdpResult{}, err
	}
	result, err := importOdp(params)
	return result, classify(err)
}
