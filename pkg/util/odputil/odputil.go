// Package odputil reads an OpenDocument presentation (.odp, and its
// template .otp) into a .qslide, the same JSON pptxutil's import writes, so
// an Impress deck opens in the read-only presentation viewer and can be
// brought into the editor (#3273, docs/architecture/read-only-documents.md).
//
// An .odp is a zip of XML parts. The package is opened in place through
// officezip, and three parts are read: meta.xml for the title, styles.xml for
// the shared styles, the page layout that sets the slide size and the master
// pages, and content.xml for the slides. Every slide is a draw:page in
// content.xml; they are streamed one at a time, and each slide is written to
// the output before the next one is read. What is held across slides is the
// style tables, bounded by [MaxStyleDefinitions] and [MaxStyleBytes], and
// each master page's own shapes, which every slide that uses the master
// draws behind its own.
//
// What carries across: text frames, with their paragraphs, alignment, line
// spacing, bullets and numbering, runs (bold, italic, underline,
// strikethrough, size, font, color), vertical anchor, padding and auto-fit;
// pictures, with their alt text; custom shapes and the plain draw:rect,
// draw:ellipse and draw:circle, as the rectangles, rounded rectangles,
// ellipses, triangles, diamonds, arrows and stars the editor has, with their
// fill, outline and opacity, and any text they hold as a text box over them;
// lines and connectors, with arrowheads; groups, nested; tables, with their
// column widths, row heights, cell text, fills, borders and merged cells;
// rotation; stacking order; slide backgrounds, a color or a picture, the
// master's where the slide sets none; transitions with a match in the editor;
// and speaker notes.
//
// What the editor has no model for is skipped, or approximated, and named in
// the slide's warnings, never a failure: charts and other embedded objects,
// video and audio, 3D scenes, form controls, animations, measurement lines
// and callouts, gradients and hatches (drawn in one color), and freeform
// shapes (drawn as rectangles). Nothing a document holds is fetched: a
// picture linked from outside the package is left out with a warning. An
// import writes schema version 1, every color literal and no theme, as
// pptxutil's does, which the editor migrates as it opens the file.
//
// # Units
//
// The slide size comes from the first master page's page layout. A 16:9 or
// 4:3 page becomes the editor's 1920×1080 or 1024×768 preset, and any other
// shape is 1920 units wide and as tall as its aspect makes it. Every length,
// stroke width and font size is scaled by the same factor, so a deck keeps
// its proportions, and is kept to two decimals.
package odputil

import (
	"errors"
	"io"
)

// Import limits. An .odp is user-supplied, and a zip says what it holds
// before it is read, so each is checked against what is actually read.
const (
	// MaxImportEntries bounds the entries of one package, checked against
	// its directory before the directory is read.
	MaxImportEntries = 20_000
	// MaxImportBytes bounds what a package unpacks to, every part together.
	MaxImportBytes = 2 << 30 // 2 GiB
	// MaxXMLPartBytes bounds one XML part unpacked. content.xml holds every
	// slide, and is streamed rather than held.
	MaxXMLPartBytes = 256 << 20 // 256 MiB
	// MaxXMLElements bounds the elements of one XML part.
	MaxXMLElements = 4_000_000
	// MaxXMLDepth bounds how deeply a part's elements nest.
	MaxXMLDepth = 256
	// MaxStyleDefinitions bounds the styles, list styles, fill images,
	// gradients, dashes and font faces held from styles.xml and content.xml
	// together.
	MaxStyleDefinitions = 20_000
	// MaxStyleBytes bounds the names and property values those definitions
	// hold.
	MaxStyleBytes = 8 << 20 // 8 MiB
	// MaxSlides is the number of slides one import writes.
	MaxSlides = 5_000
	// MaxElements bounds the elements of the whole deck, grouped ones, table
	// cells and every slide's copy of its master's shapes included.
	MaxElements = 200_000
	// MaxGroupDepth bounds how deeply groups nest.
	MaxGroupDepth = 32
	// MaxSlideTextBytes bounds the text one slide holds, its notes included:
	// a slide is held whole until it is written.
	MaxSlideTextBytes = 16 << 20 // 16 MiB
	// MaxImageBytes is the largest picture stored. A larger one is left out
	// with a warning.
	MaxImageBytes = 100 << 20 // 100 MiB
	// MaxMediaBytes bounds the pictures of one import together; past it, the
	// rest are left out with a warning.
	MaxMediaBytes = 1 << 30 // 1 GiB
)

var (
	// ErrNotOdp reports a source that is not an OpenDocument presentation:
	// not a zip, a zip with unsafe entry names, another kind of document, or
	// malformed XML. The caller's file is at fault, so callers answer 400.
	ErrNotOdp = errors.New("odputil: not an odp")
	// ErrEncrypted reports a presentation saved with a password: its
	// content cannot be read without it. Also the caller's file.
	ErrEncrypted = errors.New("odputil: the presentation is password-protected")
	// ErrTooLarge reports a presentation past one of the limits above. Also
	// the caller's file, and also a 400.
	ErrTooLarge = errors.New("odputil: presentation exceeds the size limits")
)

// StoreMediaFunc stores a picture an import found, under a name like
// image1.png whose extension matches its content, and returns the reference
// the .qslide names it by. It is the same contract pptxutil.StoreMediaFunc
// has: r fails with [ErrTooLarge] past [MaxImageBytes], and a picture that
// fails that way is skipped with a warning; any other error fails the
// import. It may return without reading r, as a viewer that serves the
// picture from the package later does.
type StoreMediaFunc func(name string, r io.Reader) (string, error)

// ImportOdpParams is one .odp on its way to becoming a .qslide.
type ImportOdpParams struct {
	// Source is the package, read in place: a zip's directory is at its
	// end.
	Source io.ReaderAt
	// Size is Source's length in bytes.
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
// the result's warnings, and is never a failure. Errors wrap [ErrNotOdp]
// when the source is not an OpenDocument presentation, [ErrEncrypted] when
// it is password-protected, and [ErrTooLarge] past the import limits;
// anything else is a read, write or StoreMedia failure. Out may hold part of
// a .qslide when an error is returned.
func ImportOdp(params ImportOdpParams) (ImportOdpResult, error) {
	if params.Source == nil || params.Out == nil {
		return ImportOdpResult{}, errors.New("odputil: Source and Out are required")
	}
	im, err := openImporter(params)
	if err != nil {
		return ImportOdpResult{}, err
	}
	return im.run()
}
