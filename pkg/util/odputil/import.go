package odputil

// cspell:ignore otp smil

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// The parts an import reads.
const (
	partMimetype = "mimetype"
	partContent  = "content.xml"
	partStyles   = "styles.xml"
	partMeta     = "meta.xml"
)

// presentationTypes are the media types an .odp and an .otp declare.
var presentationTypes = map[string]bool{
	"application/vnd.oasis.opendocument.presentation":          true,
	"application/vnd.oasis.opendocument.presentation-template": true,
}

// chartType is the media type of an embedded chart.
const chartType = "application/vnd.oasis.opendocument.chart"

// The slide size an import falls back to, Impress's 16:9 default, in cm,
// and the page sizes it accepts, one inch to 56 inches as PowerPoint does.
const (
	defaultPageWidthCm  = 28
	defaultPageHeightCm = 15.75
	minPageCm           = 2.54
	maxPageCm           = 142.24
)

// maxTitleBytes bounds the title read from meta.xml.
const maxTitleBytes = 1_000

// importer carries the state of one import across its slides.
type importer struct {
	pkg    *officezip.Package
	params ImportOdpParams
	styles *styleSet
	// charts names the embedded objects the manifest says are charts.
	charts map[string]bool
	// scale is cm per slide unit; see the package doc.
	scale     float64
	size      outSize
	sizeKnown bool
	// masters holds each master page's own shapes and background, read
	// once and drawn behind every slide that uses it.
	masters     map[string]*masterPage
	firstMaster string
	// media maps a picture part to the reference it was stored under, ""
	// when it was skipped, and skipped to why.
	media      map[string]string
	skipped    map[string]string
	mediaBytes int64
	elements   int
	nextID     int
	result     ImportOdpResult
}

// masterPage is a style:master-page as its slides draw it.
type masterPage struct {
	elements   []outElement
	background *outBackground
	warnings   []string
}

// openImporter opens the package and checks that it is a presentation
// whose content can be read.
func openImporter(params ImportOdpParams) (*importer, error) {
	pkg, err := officezip.Open(params.Source, params.Size, officezip.Limits{
		MaxEntries:    MaxImportEntries,
		MaxPartBytes:  MaxXMLPartBytes,
		MaxTotalBytes: MaxImportBytes,
		MaxElements:   MaxXMLElements,
		MaxDepth:      MaxXMLDepth,
	})
	if err != nil {
		return nil, xmlError(err)
	}
	im := &importer{
		pkg: pkg, params: params, styles: newStyleSet(), charts: map[string]bool{},
		masters: map[string]*masterPage{}, media: map[string]string{}, skipped: map[string]string{},
	}
	if pkg.Has(partMimetype) {
		mime, err := im.mimetype()
		if err != nil {
			return nil, err
		}
		if !presentationTypes[mime] {
			return nil, fmt.Errorf("%w: the package is a %q, not a presentation", ErrNotOdp, mime)
		}
	}
	if !pkg.Has(partContent) {
		return nil, fmt.Errorf("%w: no %s", ErrNotOdp, partContent)
	}
	if err := im.readManifest(); err != nil {
		return nil, err
	}
	return im, nil
}

// mimetype reads the package's declared media type.
func (im *importer) mimetype() (string, error) {
	rc, err := im.pkg.Part(partMimetype)
	if err != nil {
		return "", xmlError(err)
	}
	defer rc.Close()
	var b strings.Builder
	if _, err := io.Copy(&b, io.LimitReader(rc, 256)); err != nil {
		return "", xmlError(err)
	}
	return strings.TrimSpace(b.String()), nil
}

// readManifest notes which embedded objects are charts, and refuses a
// package whose content is encrypted. A manifest that cannot be read says
// neither.
func (im *importer) readManifest() error {
	if !im.pkg.Has("META-INF/manifest.xml") {
		return nil
	}
	entries, err := im.pkg.Manifest()
	if errors.Is(err, officezip.ErrTooLarge) {
		return xmlError(err)
	}
	for _, e := range entries {
		path := strings.TrimSuffix(strings.TrimPrefix(e.FullPath, "./"), "/")
		if strings.EqualFold(path, partContent) && e.Encrypted {
			return ErrEncrypted
		}
		if e.MediaType == chartType && len(im.charts) < MaxImportEntries {
			im.charts[path] = true
		}
	}
	return nil
}

// run converts the presentation: the title, the styles and masters, then
// every slide in show order, writing each as it goes.
func (im *importer) run() (ImportOdpResult, error) {
	title := im.title()
	if err := im.readStylesPart(); err != nil {
		return im.result, err
	}
	if !im.sizeKnown {
		im.setSize(defaultPageWidthCm, defaultPageHeightCm)
	}
	out := &qslideWriter{w: im.params.Out}
	if err := out.header(title, im.size); err != nil {
		return im.result, err
	}
	if err := im.readContent(out); err != nil {
		return im.result, err
	}
	return im.result, out.close()
}

// title is meta.xml's dc:title, or the caller's fallback.
func (im *importer) title() string {
	if !im.pkg.Has(partMeta) {
		return im.params.Title
	}
	dec, err := im.pkg.XML(partMeta)
	if err != nil {
		return im.params.Title
	}
	defer dec.Close()
	var title string
	for title == "" {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if start, ok := tok.(xml.StartElement); ok && is(start, nsDC, "title") {
			t, err := text(dec, maxTitleBytes)
			if err != nil {
				break
			}
			title = strings.TrimSpace(strings.ToValidUTF8(t, ""))
			if title == "" {
				break
			}
		}
	}
	if title == "" {
		return im.params.Title
	}
	return title
}

// setSize picks the slide units for a page w by h cm; see the package doc.
// A missing or nonsensical size is Impress's 16:9 default.
func (im *importer) setSize(w, h float64) {
	if w < minPageCm || h < minPageCm || w > maxPageCm || h > maxPageCm {
		w, h = defaultPageWidthCm, defaultPageHeightCm
	}
	aspect := w / h
	size := outSize{Width: 1920, Height: round(1920 / aspect)}
	switch {
	case math.Abs(aspect-16.0/9) < 0.01:
		size.Height = 1080
	case math.Abs(aspect-4.0/3) < 0.01:
		size = outSize{Width: 1024, Height: 768}
	}
	im.size, im.scale, im.sizeKnown = size, w/size.Width, true
}

// readStylesPart reads styles.xml: its font faces and styles, then its
// master pages, whose shapes are converted once here.
func (im *importer) readStylesPart() error {
	if !im.pkg.Has(partStyles) {
		return nil
	}
	dec, err := im.pkg.XML(partStyles)
	if err != nil {
		return xmlError(err)
	}
	defer dec.Close()
	if err := root(dec, partStyles, "document-styles"); err != nil {
		return err
	}
	return children(dec, func(c xml.StartElement) error {
		switch {
		case is(c, nsOffice, "font-face-decls"):
			return im.styles.readFontFaces(dec)
		case is(c, nsOffice, "styles"):
			return im.styles.readStyles(dec, false, scopeMaster)
		case is(c, nsOffice, "automatic-styles"):
			return im.styles.readStyles(dec, true, scopeMaster)
		case is(c, nsOffice, "master-styles"):
			return children(dec, func(m xml.StartElement) error {
				if !is(m, nsStyle, "master-page") {
					return skip(dec)
				}
				return im.readMaster(dec, m)
			})
		}
		return skip(dec)
	})
}

// readMaster reads one style:master-page: its page layout, which sets the
// slide size when it is the first, its background and its own shapes. Its
// placeholders are only the slides' templates, and are left out.
func (im *importer) readMaster(dec *officezip.Decoder, start xml.StartElement) error {
	name := attr(start, nsStyle, "name")
	if err := im.styles.define(name); err != nil {
		return err
	}
	if !im.sizeKnown {
		if wh, ok := im.styles.pageLayouts[attr(start, nsStyle, "page-layout-name")]; ok {
			im.setSize(wh[0], wh[1])
		} else {
			im.setSize(defaultPageWidthCm, defaultPageHeightCm)
		}
	}
	s := im.newSlideReader(scopeMaster)
	s.master = true
	m := &masterPage{}
	err := children(dec, func(c xml.StartElement) error {
		if is(c, nsPresentation, "notes") {
			return skip(dec)
		}
		els, err := s.element(dec, c, 0)
		m.elements = append(m.elements, els...)
		return err
	})
	if err != nil {
		return err
	}
	if m.background, _, err = s.background(attr(start, nsDraw, "style-name")); err != nil {
		return err
	}
	m.warnings = s.warnings
	if _, dup := im.masters[name]; !dup {
		im.masters[name] = m
	}
	if im.firstMaster == "" {
		im.firstMaster = name
	}
	return nil
}

// readContent streams content.xml: its font faces and automatic styles,
// held for the slides, then each draw:page, written as it is read.
func (im *importer) readContent(out *qslideWriter) error {
	dec, err := im.pkg.XML(partContent)
	if err != nil {
		return xmlError(err)
	}
	defer dec.Close()
	if err := root(dec, partContent, "document-content"); err != nil {
		return err
	}
	found := false
	err = children(dec, func(c xml.StartElement) error {
		switch {
		case is(c, nsOffice, "font-face-decls"):
			return im.styles.readFontFaces(dec)
		case is(c, nsOffice, "automatic-styles"):
			return im.styles.readStyles(dec, true, scopeContent)
		case is(c, nsOffice, "body"):
			return children(dec, func(b xml.StartElement) error {
				if !is(b, nsOffice, "presentation") {
					return skip(dec)
				}
				found = true
				return children(dec, func(p xml.StartElement) error {
					if !is(p, nsDraw, "page") {
						return skip(dec)
					}
					return im.writePage(dec, p, out)
				})
			})
		}
		return skip(dec)
	})
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: the document has no presentation body", ErrNotOdp)
	}
	return nil
}

// writePage converts one draw:page and writes it.
func (im *importer) writePage(dec *officezip.Decoder, start xml.StartElement, out *qslideWriter) error {
	number := im.result.Slides + 1
	if number > MaxSlides {
		return fmt.Errorf("%w: the presentation holds more than %d slides", ErrTooLarge, MaxSlides)
	}
	slide, warnings, err := im.readPage(dec, start)
	if err != nil {
		return err
	}
	slide.ID = fmt.Sprintf("s%d", number)
	for _, w := range warnings {
		im.result.Warnings = append(im.result.Warnings, ImportWarning{Slide: number, Message: w})
	}
	if err := out.slide(slide); err != nil {
		return err
	}
	im.result.Slides++
	return nil
}

// readPage converts one draw:page: its master's shapes, then its own, then
// its background, transition and notes.
func (im *importer) readPage(dec *officezip.Decoder, start xml.StartElement) (outSlide, []string, error) {
	s := im.newSlideReader(scopeContent)
	pageStyle := attr(start, nsDraw, "style-name")
	pageRefs := []styleRef{{"drawing-page", pageStyle}}
	master := im.masters[attr(start, nsDraw, "master-page-name")]
	if master == nil {
		master = im.masters[im.firstMaster]
	}
	slide := outSlide{Elements: []outElement{}}
	if v, _ := s.styles.prop(s.scope, pageRefs, "presentation:background-objects-visible"); master != nil && v != "false" {
		for _, el := range master.elements {
			copied, err := s.reID(el)
			if err != nil {
				return outSlide{}, nil, err
			}
			slide.Elements = append(slide.Elements, copied)
		}
		for _, w := range master.warnings {
			s.warn(w)
		}
	}
	err := children(dec, func(c xml.StartElement) error {
		switch {
		case is(c, nsPresentation, "notes"):
			notes, err := s.notes(dec)
			slide.Notes = notes
			return err
		case is(c, nsAnim, "par"), is(c, nsAnim, "seq"):
			return s.animations(dec)
		}
		els, err := s.element(dec, c, 0)
		slide.Elements = append(slide.Elements, els...)
		return err
	})
	if err != nil {
		return outSlide{}, nil, err
	}
	bg, set, err := s.background(pageStyle)
	if err != nil {
		return outSlide{}, nil, err
	}
	if v, _ := s.styles.prop(s.scope, pageRefs, "presentation:background-visible"); !set && master != nil && v != "false" {
		bg = master.background
	}
	slide.Background = bg
	slide.Transition = s.transition(pageRefs)
	return slide, s.warnings, nil
}

// newSlideReader starts converting one slide, or one master page.
func (im *importer) newSlideReader(scope int) *slideReader {
	return &slideReader{importer: im, scope: scope, warned: map[string]bool{}}
}

// slideReader converts one slide's or master page's shapes.
type slideReader struct {
	*importer
	scope int
	// master is set while a master page's own shapes are read.
	master    bool
	warnings  []string
	warned    map[string]bool
	textBytes int
	// transitionMs is the duration the slide's timing tree gives its
	// transition, 0 when it gives none.
	transitionMs int
}

// warn records a warning once per slide.
func (s *slideReader) warn(message string) {
	if !s.warned[message] {
		s.warned[message] = true
		s.warnings = append(s.warnings, message)
	}
}

// id returns the next element id.
func (s *slideReader) id() string {
	s.nextID++
	return fmt.Sprintf("e%d", s.nextID)
}

// countElement charges one element against [MaxElements].
func (s *slideReader) countElement() error {
	s.elements++
	if s.elements > MaxElements {
		return fmt.Errorf("%w: the presentation holds more than %d elements", ErrTooLarge, MaxElements)
	}
	return nil
}

// countText charges text against the slide's [MaxSlideTextBytes].
func (s *slideReader) countText(n int) error {
	s.textBytes += n
	if s.textBytes > MaxSlideTextBytes {
		return fmt.Errorf("%w: a slide holds more than %d bytes of text", ErrTooLarge, MaxSlideTextBytes)
	}
	return nil
}

// units converts a length in cm to slide units.
func (s *slideReader) units(cm float64) float64 {
	return round(math.Max(-maxCm, math.Min(maxCm, cm)) / s.scale)
}

// animations reads a slide's anim:par timing tree, which holds its
// animations, and warns when it holds any: a timing tree with nothing but
// its empty sequences, or only the slide's transition, animates nothing.
func (s *slideReader) animations(dec *officezip.Decoder) error {
	return children(dec, func(c xml.StartElement) error {
		switch {
		case is(c, nsAnim, "par"), is(c, nsAnim, "seq"):
			return s.animations(dec)
		case is(c, nsAnim, "transitionFilter"):
			s.transitionMs = parseDuration(attr(c, nsSmil, "dur"))
			return skip(dec)
		}
		s.warn("Animations are not imported.")
		return skip(dec)
	})
}
