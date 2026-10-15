package docreadutil

// cspell:ignore descr blip docPr graphicData imagedata txbx txbxContent

import (
	"encoding/xml"
	"strconv"
	"strings"
)

// emuPerPixel is how many English Metric Units make a CSS pixel at 96 dpi.
const emuPerPixel = 9525

// maxImagePixels bounds an image's declared width or height.
const maxImagePixels = 20_000

// scanDrawing reads a w:drawing through to its end, and says what it holds.
// A text box's paragraphs inside it are skipped with the rest.
func scanDrawing(src tokenSource) (drawingInfo, error) {
	var info drawingInfo
	uri := ""
	picture, hasText, hasChart := false, false, false
	err := scan(src, func(t xml.StartElement) {
		switch t.Name.Local {
		case "extent":
			if info.width == 0 && info.height == 0 {
				info.width = emuToPixels(attr(t, "cx"))
				info.height = emuToPixels(attr(t, "cy"))
			}
		case "docPr":
			if info.alt == "" {
				info.alt = firstNonEmpty(attr(t, "descr"), attr(t, "title"))
			}
		case "graphicData":
			if uri == "" {
				uri = attr(t, "uri")
			}
		case "pic":
			picture = true
		case "blip":
			if info.embed == "" && info.link == "" {
				info.embed, info.link = attr(t, "embed"), attr(t, "link")
			}
		case "chart":
			hasChart = true
		case "txbx", "txbxContent", "textbox":
			hasText = true
		}
	})
	switch {
	case err != nil:
		return drawingInfo{}, err
	case strings.HasSuffix(uri, "/chart") || hasChart:
		info.what = "chart"
	case strings.HasSuffix(uri, "/diagram"):
		info.what = "diagram"
	case picture && (info.embed != "" || info.link != ""):
	case picture:
		info.what = "image"
	case hasText:
		info.what = "text box"
	default:
		info.what = "shape"
	}
	return info, nil
}

// scanVML reads a w:pict, a picture or shape in the older VML, through to
// its end, and says what it holds.
func scanVML(src tokenSource) (drawingInfo, error) {
	var info drawingInfo
	hasText := false
	err := scan(src, func(t xml.StartElement) {
		switch t.Name.Local {
		case "imagedata":
			if info.embed == "" {
				info.embed = attr(t, "id")
			}
			if info.alt == "" {
				info.alt = attr(t, "title")
			}
		case "textbox":
			hasText = true
		default:
			if info.alt == "" {
				info.alt = attr(t, "alt")
			}
			if style := attr(t, "style"); style != "" && info.width == 0 && info.height == 0 {
				info.width, info.height = vmlSize(style)
			}
		}
	})
	switch {
	case err != nil:
		return drawingInfo{}, err
	case info.embed != "":
	case hasText:
		info.what = "text box"
	default:
		info.what = "shape"
	}
	return info, nil
}

// scan reads the element whose start was just read through to its end,
// handing every descendant's start to fn.
func scan(src tokenSource, fn func(xml.StartElement)) error {
	for depth := 1; depth > 0; {
		tok, err := src.Token()
		if err != nil {
			return xmlError(err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			fn(t)
		case xml.EndElement:
			depth--
		}
	}
	return nil
}

// emuToPixels converts a length in EMU to CSS pixels, 0 when it is not a
// sensible length.
func emuToPixels(s string) int {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return int(min(n/emuPerPixel, maxImagePixels))
}

// vmlSize reads the width and height from a VML style attribute, such as
// "width:120pt;height:80pt", in CSS pixels.
func vmlSize(style string) (width, height int) {
	for _, decl := range strings.Split(style, ";") {
		name, value, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(name)) {
		case "width":
			width = cssPixels(value)
		case "height":
			height = cssPixels(value)
		}
	}
	return width, height
}

// cssPixels converts a CSS length in pt, in, cm, mm or px to pixels, 0 when
// it is none of those.
func cssPixels(value string) int {
	value = strings.TrimSpace(strings.ToLower(value))
	units := map[string]float64{"pt": 96.0 / 72, "in": 96, "cm": 96 / 2.54, "mm": 96 / 25.4, "px": 1}
	for unit, scale := range units {
		if number, ok := strings.CutSuffix(value, unit); ok {
			f, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
			if err != nil || f <= 0 || f > maxImagePixels {
				return 0
			}
			return min(int(f*scale), maxImagePixels)
		}
	}
	return 0
}

// firstNonEmpty returns the first of values that is not empty.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
