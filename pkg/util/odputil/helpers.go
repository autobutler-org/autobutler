package odputil

// cspell:ignore smil xlink

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// The ODF namespaces the reader matches names in.
const (
	nsOffice       = "urn:oasis:names:tc:opendocument:xmlns:office:1.0"
	nsStyle        = "urn:oasis:names:tc:opendocument:xmlns:style:1.0"
	nsText         = "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
	nsTable        = "urn:oasis:names:tc:opendocument:xmlns:table:1.0"
	nsDraw         = "urn:oasis:names:tc:opendocument:xmlns:drawing:1.0"
	nsFo           = "urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
	nsXlink        = "http://www.w3.org/1999/xlink"
	nsDC           = "http://purl.org/dc/elements/1.1/"
	nsPresentation = "urn:oasis:names:tc:opendocument:xmlns:presentation:1.0"
	nsSVG          = "urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0"
	nsAnim         = "urn:oasis:names:tc:opendocument:xmlns:animation:1.0"
	nsSmil         = "urn:oasis:names:tc:opendocument:xmlns:smil-compatible:1.0"
	nsDr3d         = "urn:oasis:names:tc:opendocument:xmlns:dr3d:1.0"
	nsForm         = "urn:oasis:names:tc:opendocument:xmlns:form:1.0"
)

// nsPrefixes maps the namespaces a style property can be in to the prefix
// it is held under, so draw:fill and fo:padding never meet.
var nsPrefixes = map[string]string{
	nsStyle: "style", nsText: "text", nsTable: "table", nsDraw: "draw", nsFo: "fo",
	nsPresentation: "presentation", nsSVG: "svg", nsSmil: "smil",
}

// cmPer converts a length unit to centimeters.
var cmPer = map[string]float64{
	"cm": 1, "mm": 0.1, "in": 2.54, "inch": 2.54, "pt": 2.54 / 72, "pc": 2.54 / 6, "px": 2.54 / 96,
}

// lengthPattern is an ODF length: a number and its unit.
var lengthPattern = regexp.MustCompile(`^\s*([-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)\s*([a-z]*)\s*$`)

// hexColorPattern is an ODF color, #rrggbb.
var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// maxCm bounds any length read, in centimeters: ten meters is past any
// slide, and keeps every later sum finite.
const maxCm = 1_000

// is reports whether the element is ns:local.
func is(start xml.StartElement, ns, local string) bool {
	return start.Name.Space == ns && start.Name.Local == local
}

// attr returns the value of the element's ns:local attribute.
func attr(start xml.StartElement, ns, local string) string {
	for _, a := range start.Attr {
		if a.Name.Local == local && a.Name.Space == ns {
			return a.Value
		}
	}
	return ""
}

// children reads the children of the element whose start was just read,
// through to its end. fn gets each child element's start and must read the
// child through to its end, by reading its children or skipping it.
func children(dec *officezip.Decoder, fn func(xml.StartElement) error) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return xmlError(err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if err := fn(t); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

// skip reads the rest of the element whose start was just read.
func skip(dec *officezip.Decoder) error {
	return xmlError(dec.Skip())
}

// text reads the character data of the element whose start was just read,
// through to its end and at most limit bytes of it, skipping any child
// elements.
func text(dec *officezip.Decoder, limit int) (string, error) {
	var sb strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", xmlError(err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			if room := limit - sb.Len(); room > 0 {
				sb.Write(t[:min(len(t), room)])
			}
		case xml.StartElement:
			if err := skip(dec); err != nil {
				return "", err
			}
		case xml.EndElement:
			return sb.String(), nil
		}
	}
}

// root reads up to a part's root element and checks its name.
func root(dec *officezip.Decoder, part, local string) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return xmlError(err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			if !is(start, nsOffice, local) {
				return fmt.Errorf("%w: %s is not an office:%s part", ErrNotOdp, part, local)
			}
			return nil
		}
	}
}

// xmlError makes an error from officezip or encoding/xml one ImportOdp
// reports: the package's limits as [ErrTooLarge], and a malformed part, or
// one that ends before its elements do, as [ErrNotOdp].
func xmlError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrTooLarge), errors.Is(err, ErrNotOdp):
		return err
	case errors.Is(err, officezip.ErrTooLarge):
		return fmt.Errorf("%w: %v", ErrTooLarge, err)
	case errors.Is(err, io.EOF):
		return fmt.Errorf("%w: a part ends before its elements do", ErrNotOdp)
	}
	return fmt.Errorf("%w: %v", ErrNotOdp, err)
}

// parseCm reads an ODF length in centimeters. A length without a unit, in
// a unit this reader does not know, or past ten meters is not one.
func parseCm(v string) (float64, bool) {
	m := lengthPattern.FindStringSubmatch(v)
	if m == nil {
		return 0, false
	}
	per, ok := cmPer[m[2]]
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, false
	}
	cm := n * per
	if math.Abs(cm) > maxCm {
		return 0, false
	}
	return cm, true
}

// parsePercent reads a percentage, "50%", as a fraction.
func parsePercent(v string) (float64, bool) {
	s, ok := strings.CutSuffix(strings.TrimSpace(v), "%")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 10_000 {
		return 0, false
	}
	return n / 100, true
}

// parseColor reads an ODF color as .qslide writes one, #RRGGBB, or "" when
// it is not one.
func parseColor(v string) string {
	v = strings.TrimSpace(v)
	if !hexColorPattern.MatchString(v) {
		return ""
	}
	return strings.ToUpper(v)
}

// isTrue reads an ODF boolean.
func isTrue(v string) bool { return strings.TrimSpace(v) == "true" }
