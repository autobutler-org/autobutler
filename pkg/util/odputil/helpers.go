package odputil

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// canceled reports whether ctx is already done.
func canceled(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// classify turns a package or parse error into one of this package's
// sentinels. Context cancellation, and a write or StoreMedia failure, is
// returned as itself.
func classify(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, officezip.ErrMissingPart) && !errors.Is(err, ErrMalformed) {
		return fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	return err
}

// readNode reads the element start opened, and everything in it, as a
// node, charging each node and its bytes to budget.
func readNode(dec *officezip.Decoder, start xml.StartElement, budget *nodeBudget) (*node, error) {
	root := &node{name: start.Name, attrs: start.Copy().Attr}
	if err := budget.charge(root); err != nil {
		return nil, err
	}
	stack := []*node{root}
	for len(stack) > 0 {
		tok, err := dec.Token()
		if err != nil {
			return nil, xmlError(err)
		}
		top := stack[len(stack)-1]
		switch t := tok.(type) {
		case xml.StartElement:
			child := &node{name: t.Name, attrs: t.Copy().Attr}
			if err := budget.charge(child); err != nil {
				return nil, err
			}
			top.children = append(top.children, child)
			stack = append(stack, child)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			text := &node{text: string(t)}
			if err := budget.charge(text); err != nil {
				return nil, err
			}
			top.children = append(top.children, text)
		}
	}
	return root, nil
}

// xmlError keeps officezip's own errors and makes any other decode failure
// the file's.
func xmlError(err error) error {
	if errors.Is(err, ErrTooLarge) || errors.Is(err, ErrMalformed) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrMalformed, err)
}

// charge takes one node and its bytes from the budget.
func (b *nodeBudget) charge(n *node) error {
	b.nodes--
	b.bytes -= len(n.text) + len(n.name.Local)
	for _, a := range n.attrs {
		b.bytes -= len(a.Name.Local) + len(a.Value)
	}
	if b.nodes < 0 || b.bytes < 0 {
		return fmt.Errorf("%w: %s", ErrTooLarge, b.what)
	}
	return nil
}

// is reports whether n is the element local in namespace space.
func (n *node) is(space, local string) bool {
	return n.name.Local == local && n.name.Space == space
}

// attr is n's attribute local in namespace space, or "".
func (n *node) attr(space, local string) string {
	for _, a := range n.attrs {
		if a.Name.Local == local && a.Name.Space == space {
			return a.Value
		}
	}
	return ""
}

// child is n's first child element local in namespace space, or nil.
func (n *node) child(space, local string) *node {
	for _, c := range n.children {
		if c.is(space, local) {
			return c
		}
	}
	return nil
}

// innerText is every text node under n, in order.
func (n *node) innerText() string {
	var b strings.Builder
	var walk func(*node)
	walk = func(n *node) {
		b.WriteString(n.text)
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// parseCM reads an ODF length — 2.5cm, 25mm, 1in, 72pt, 6pc, 96px — in
// centimeters. A bare or unknown unit is not a length.
func parseCM(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	units := []struct {
		suffix string
		cm     float64
	}{
		{"cm", 1}, {"mm", 0.1}, {"in", 2.54}, {"pt", 2.54 / 72}, {"pc", 2.54 / 6}, {"px", 2.54 / 96},
	}
	for _, u := range units {
		if v, ok := strings.CutSuffix(s, u.suffix); ok {
			n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1e6 {
				return 0, false
			}
			return n * u.cm, true
		}
	}
	return 0, false
}

// length reads an ODF length in slide units.
func (im *importer) length(s string) (float64, bool) {
	cm, ok := parseCM(s)
	return cm * im.unit, ok
}

// parsePercent reads 50% as 0.5.
func parsePercent(s string) (float64, bool) {
	v, ok := strings.CutSuffix(strings.TrimSpace(s), "%")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, false
	}
	return n / 100, true
}

// parseColor reads an ODF color, #rrggbb, as the .qslide writes it:
// #RRGGBB. Anything else is no color.
func parseColor(s string) string {
	s = strings.TrimSpace(s)
	if len(s) != 7 || s[0] != '#' {
		return ""
	}
	if _, err := strconv.ParseUint(s[1:], 16, 32); err != nil {
		return ""
	}
	return strings.ToUpper(s)
}

// withAlpha appends an opacity from 0 to 1 to a #RRGGBB color, left out
// when opaque.
func withAlpha(c string, opacity float64) string {
	a := int(math.Round(math.Max(0, math.Min(1, opacity)) * 255))
	if c == "" || a >= 255 {
		return c
	}
	return fmt.Sprintf("%s%02X", c, a)
}

// round keeps two decimals of a converted length, and never writes -0.
func round(v float64) float64 {
	r := math.Round(v*100) / 100
	if r == 0 {
		return 0
	}
	return r
}

// mediaName is the name a picture is stored under: its own stem, kept to
// characters every filesystem takes, and the extension of its content.
func mediaName(part, ext string) string {
	stem := strings.TrimSuffix(path.Base(part), path.Ext(part))
	var b strings.Builder
	for _, r := range stem {
		if b.Len() >= 64 {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		b.WriteString("image")
	}
	return b.String() + "." + ext
}

// fontFamily strips the quotes a font family may be written in.
func fontFamily(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return strings.Trim(s, `'"`)
}

// atoiDefault reads a non-negative integer attribute, def when it is absent
// or not one.
func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return def
	}
	return n
}
