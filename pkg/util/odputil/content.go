package odputil

// cspell:ignore smil xlink

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// The page size a presentation that names none is drawn at: LibreOffice's
// 16:9 default, 28 cm by 15.75 cm.
const (
	defaultPageWidthCM  = 28
	defaultPageHeightCM = 15.75
	// minPageCM and maxPageCM bound a page side that is taken as written,
	// PowerPoint's own bounds: 1 in to 56 in.
	minPageCM = 2.54
	maxPageCM = 142.24
)

// importOdp opens the package, reads its styles, and streams content.xml a
// slide at a time.
func importOdp(params ImportOdpParams) (ImportOdpResult, error) {
	pkg, err := officezip.Open(params.Source, params.Size, officezip.Limits{
		MaxPartBytes:  MaxPartBytes,
		MaxTotalBytes: MaxImportBytes,
	})
	if err != nil {
		return ImportOdpResult{}, err
	}
	if err := checkPackage(pkg); err != nil {
		return ImportOdpResult{}, err
	}
	im := &importer{
		pkg:     pkg,
		params:  params,
		styles:  newStyleSet(),
		budget:  &nodeBudget{nodes: maxStyleNodes, bytes: maxStyleBytes, what: "the styles and master pages are too large"},
		media:   map[string]string{},
		skipped: map[string]string{},
	}
	if err := im.readStyles(); err != nil {
		return im.result, err
	}
	title := readTitle(pkg)
	if title == "" {
		title = params.Title
	}
	if err := canceled(params.Ctx); err != nil {
		return im.result, err
	}
	return im.result, im.readContent(title)
}

// readContent streams content.xml: its fonts and automatic styles are held,
// and each draw:page is converted and written as it is read.
func (im *importer) readContent(title string) error {
	dec, err := im.pkg.XML(contentPart)
	if err != nil {
		return err
	}
	defer dec.Close()
	var out *qslideWriter
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return xmlError(err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch {
		case start.Name.Space == nsOffice && (start.Name.Local == "document-content" || start.Name.Local == "body"):
			continue
		case start.Name.Space == nsOffice && start.Name.Local == "presentation" && out == nil:
			// The header waits for the presentation, so a file that is some
			// other kind of document fails before the first byte.
			out = &qslideWriter{w: im.params.Out}
			if err := out.header(title, im.setSize()); err != nil {
				return err
			}
			continue
		case start.Name.Space == nsOffice && (start.Name.Local == "font-face-decls" || start.Name.Local == "automatic-styles"):
			n, err := readNode(dec, start, im.budget)
			if err != nil {
				return err
			}
			im.styles.add(n, scopeContent)
			continue
		case start.Name.Space == nsDraw && start.Name.Local == "page" && out != nil:
			if err := im.page(dec, start, out); err != nil {
				return err
			}
			continue
		}
		if err := dec.Skip(); err != nil {
			return xmlError(err)
		}
	}
	if out == nil {
		return fmt.Errorf("%w: content.xml holds no presentation", ErrMalformed)
	}
	return out.close()
}

// page reads one draw:page, converts it and writes it.
func (im *importer) page(dec *officezip.Decoder, start xml.StartElement, out *qslideWriter) error {
	if err := canceled(im.params.Ctx); err != nil {
		return err
	}
	number := im.result.Slides + 1
	if number > MaxSlides {
		return fmt.Errorf("%w: the presentation holds more than %d slides", ErrTooLarge, MaxSlides)
	}
	budget := &nodeBudget{nodes: maxPageNodes, bytes: maxPageBytes, what: fmt.Sprintf("slide %d is too large", number)}
	n, err := readNode(dec, start, budget)
	if err != nil {
		return err
	}
	slide, warnings, err := im.convertPage(n)
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

// setSize picks the slide units for the first master's page; see the
// package doc. A missing or nonsensical size is LibreOffice's 16:9 default.
func (im *importer) setSize() outSize {
	w, h := float64(defaultPageWidthCM), defaultPageHeightCM
	if m := im.styles.master(""); m != nil {
		if layout, ok := im.styles.layouts[m.layout]; ok {
			w, h = layout.width, layout.height
		}
	}
	if w < minPageCM || h < minPageCM || w > maxPageCM || h > maxPageCM {
		w, h = defaultPageWidthCM, defaultPageHeightCM
	}
	aspect := w / h
	size := outSize{Width: 1920, Height: round(1920 / aspect)}
	switch {
	case math.Abs(aspect-16.0/9) < 0.01:
		size.Height = 1080
	case math.Abs(aspect-4.0/3) < 0.01:
		size = outSize{Width: 1024, Height: 768}
	}
	im.unit = size.Width / w
	return size
}

// convertPage converts one slide: its background, its master's own shapes,
// its own shapes, its transition and its notes.
func (im *importer) convertPage(n *node) (outSlide, []string, error) {
	s := &slideReader{importer: im, scope: scopeContent, warned: map[string]bool{}}
	s.page = im.styles.master(n.attr(nsDraw, "master-page-name"))
	pageRef := []styleRef{{scope: scopeContent, family: "drawing-page", name: n.attr(nsDraw, "style-name")}}
	slide := outSlide{Elements: []outElement{}}

	var err error
	if slide.Background, err = s.background(pageRef); err != nil {
		return outSlide{}, nil, err
	}
	if s.page != nil && im.styles.get(pageRef, "drawing-page-properties", "presentation:background-objects-visible") != "false" {
		s.scope, s.master = scopeMaster, true
		children, err := s.convertChildren(s.page.shapes, 0)
		if err != nil {
			return outSlide{}, nil, err
		}
		slide.Elements = append(slide.Elements, children...)
		s.scope, s.master = scopeContent, false
	}
	var shapes []*node
	for _, c := range n.children {
		switch {
		case c.is(nsPresentation, "notes"):
			if slide.Notes, err = s.notes(c); err != nil {
				return outSlide{}, nil, err
			}
		case c.name.Space == nsAnim:
			s.warn("Animations are not imported.")
		default:
			shapes = append(shapes, c)
		}
	}
	children, err := s.convertChildren(shapes, 0)
	if err != nil {
		return outSlide{}, nil, err
	}
	slide.Elements = append(slide.Elements, children...)
	slide.Transition = s.transition(pageRef)
	return slide, s.warnings, nil
}

// background is the slide's own background, or its master's when it sets
// none and shows the master's.
func (s *slideReader) background(pageRef []styleRef) (*outBackground, error) {
	refs := [][]styleRef{pageRef}
	if s.page != nil && s.styles.get(pageRef, "drawing-page-properties", "presentation:background-visible") != "false" {
		refs = append(refs, []styleRef{{scope: scopeMaster, family: "drawing-page", name: s.page.pageStyle}})
	}
	for _, ref := range refs {
		get := func(key string) string { return s.styles.get(ref, "drawing-page-properties", key) }
		switch get("draw:fill") {
		case "solid":
			if c := parseColor(get("draw:fill-color")); c != "" {
				return &outBackground{Color: c}, nil
			}
		case "gradient", "hatch":
			s.warn("Gradient and pattern backgrounds were drawn in one color.")
			c := parseColor(get("draw:fill-color"))
			if g := s.styles.gradients[get("draw:fill-gradient-name")]; g != "" {
				c = g
			}
			if c != "" {
				return &outBackground{Color: c}, nil
			}
		case "bitmap":
			href := s.styles.fillImages[get("draw:fill-image-name")]
			if href == "" {
				continue
			}
			ref, ok, err := s.pictures([]string{href})
			if err != nil || ok {
				return &outBackground{Image: ref}, err
			}
		case "none":
			return nil, nil
		}
	}
	return nil, nil
}

// transition is the slide's transition, or nil for none or one the editor
// has no match for.
func (s *slideReader) transition(pageRef []styleRef) *outTransition {
	get := func(key string) string { return s.styles.get(pageRef, "drawing-page-properties", key) }
	kind, subtype, reverse := get("smil:type"), get("smil:subtype"), get("smil:direction") == "reverse"
	var t *outTransition
	switch kind {
	case "":
		if style := get("presentation:transition-style"); style != "" && style != "none" {
			s.warn("This slide's transition has no match in the editor, so it was left out.")
		}
		return nil
	case "fade":
		t = &outTransition{Kind: "fade"}
	case "pushWipe":
		dir := map[string]string{"fromRight": "", "fromLeft": "right", "fromTop": "down", "fromBottom": "up"}
		d, ok := dir[subtype]
		if !ok && subtype != "" {
			s.warn("This slide's transition has no match in the editor, so it was left out.")
			return nil
		}
		t = &outTransition{Kind: "push", Direction: d}
	case "barWipe":
		d := "right"
		if subtype == "topToBottom" {
			d = "down"
		}
		if reverse {
			d = map[string]string{"right": "", "down": "up"}[d]
		}
		t = &outTransition{Kind: "wipe", Direction: d}
	default:
		s.warn("This slide's transition has no match in the editor, so it was left out.")
		return nil
	}
	ms := 500
	if dur := get("smil:dur"); dur != "" {
		if v, ok := parseSeconds(dur); ok {
			ms = int(math.Round(v * 1000))
		}
	} else {
		ms = map[string]int{"slow": 1000, "medium": 500, "fast": 250}[get("presentation:transition-speed")]
	}
	ms = max(0, min(ms, 60_000))
	if ms != 500 && ms != 0 {
		t.Duration = ms
	}
	return t
}

// parseSeconds reads a SMIL clock value, 1.5s or 500ms, in seconds.
func parseSeconds(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	scale := 1.0
	switch {
	case strings.HasSuffix(s, "ms"):
		s, scale = strings.TrimSuffix(s, "ms"), 0.001
	case strings.HasSuffix(s, "s"):
		s = strings.TrimSuffix(s, "s")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0, false
	}
	return v * scale, true
}

// notes is the text of a presentation:notes page's notes frame, a line per
// paragraph.
func (s *slideReader) notes(n *node) (string, error) {
	var lines []string
	var walk func(*node) error
	walk = func(n *node) error {
		if n.is(nsDraw, "frame") && n.attr(nsPresentation, "class") == "notes" {
			if box := n.child(nsDraw, "text-box"); box != nil {
				paragraphs, err := s.paragraphs(box, graphicRefs(n, s.scope))
				if err != nil {
					return err
				}
				for _, p := range paragraphs {
					var line strings.Builder
					for _, r := range p.Runs {
						line.WriteString(r.Text)
					}
					lines = append(lines, line.String())
				}
			}
			return nil
		}
		for _, c := range n.children {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	err := walk(n)
	return strings.TrimRight(strings.Join(lines, "\n"), "\n"), err
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

// countElement charges n elements against [MaxElements].
func (s *slideReader) countElement(n int) error {
	s.elements += n
	if s.elements > MaxElements {
		return fmt.Errorf("%w: the presentation holds more than %d elements", ErrTooLarge, MaxElements)
	}
	return nil
}
