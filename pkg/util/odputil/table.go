package odputil

import (
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// Table limits, as quark_slides' TableElement has them: a table past them
// is not a table the editor opens.
const (
	maxTableRows    = 500
	maxTableColumns = 100
	maxTableCells   = 5_000
)

// defaultColumnUnits and defaultRowUnits size a column or row when neither
// it nor its frame says how big it is.
const (
	defaultColumnUnits = 100
	defaultRowUnits    = 40
)

// tableReader collects one table:table's grid as it streams past.
type tableReader struct {
	s       *slideReader
	ctx     textContext
	columns []float64
	rows    []float64
	cells   [][]outCell
	// tooBig is set once the table is past the editor's limits; the rest
	// of it is skipped.
	tooBig bool
}

// table converts a table:table inside a draw:frame. A table past the
// editor's limits is skipped with a warning.
func (s *slideReader) table(dec *officezip.Decoder, t, frameEl xml.StartElement, frame outFrame) ([]outElement, error) {
	tr := &tableReader{s: s, ctx: s.textContext(frameEl)}
	if err := tr.readGroup(dec); err != nil {
		return nil, err
	}
	if tr.tooBig {
		s.warn(fmt.Sprintf("Tables of more than %d rows, %d columns or %d cells are not imported.",
			maxTableRows, maxTableColumns, maxTableCells))
		return nil, nil
	}
	cols := len(tr.columns)
	for _, row := range tr.cells {
		cols = max(cols, len(row))
	}
	rows := len(tr.rows)
	if rows == 0 || cols == 0 {
		return nil, nil
	}
	if rows*cols > maxTableCells || cols > maxTableColumns {
		s.warn(fmt.Sprintf("Tables of more than %d rows, %d columns or %d cells are not imported.",
			maxTableRows, maxTableColumns, maxTableCells))
		return nil, nil
	}
	// Each cell is drawn as a shape is, so the deck's budget pays for it.
	for range rows * cols {
		if err := s.countElement(); err != nil {
			return nil, err
		}
	}
	for len(tr.columns) < cols {
		tr.columns = append(tr.columns, 0)
	}
	out := outTable{
		ID: s.id(), Type: typeTable, Frame: frame,
		Columns:    fillSizes(tr.columns, frame.Width, defaultColumnUnits),
		Rows:       fillSizes(tr.rows, frame.Height, 1),
		Cells:      tr.cells,
		HeaderRow:  isTrue(attr(t, nsTable, "use-first-row-styles")),
		BandedRows: isTrue(attr(t, nsTable, "use-banding-rows-styles")),
	}
	for r := range out.Cells {
		for len(out.Cells[r]) < cols {
			out.Cells[r] = append(out.Cells[r], outCell{})
		}
	}
	// The editor needs every column and row to have a size; the frame is
	// what they add up to.
	out.Frame.Width, out.Frame.Height = 0, 0
	for _, w := range out.Columns {
		out.Frame.Width += w
	}
	for _, h := range out.Rows {
		out.Frame.Height += h
	}
	out.Frame.Width, out.Frame.Height = round(out.Frame.Width), round(out.Frame.Height)
	return []outElement{out}, nil
}

// fillSizes gives every size of 0 — a column or row no style sizes, or
// one Impress sizes to fit its content — an even share of what the frame
// has left over, or fallback when it has nothing left.
func fillSizes(sizes []float64, total, fallback float64) []float64 {
	known, unknown := 0.0, 0
	for _, v := range sizes {
		if v > 0 {
			known += v
		} else {
			unknown++
		}
	}
	share := fallback
	if unknown > 0 && total-known >= 1 {
		share = (total - known) / float64(unknown)
	}
	out := make([]float64, len(sizes))
	for i, v := range sizes {
		if v <= 0 {
			v = share
		}
		out[i] = round(math.Max(v, 1))
	}
	return out
}

// readGroup reads the children of a table or of one of its column or row
// groups.
func (tr *tableReader) readGroup(dec *officezip.Decoder) error {
	return children(dec, func(c xml.StartElement) error {
		if tr.tooBig || c.Name.Space != nsTable {
			return skip(dec)
		}
		switch c.Name.Local {
		case "table-columns", "table-header-columns", "table-column-group", "table-rows", "table-header-rows",
			"table-row-group":
			return tr.readGroup(dec)
		case "table-column":
			n := repeated(attr(c, nsTable, "number-columns-repeated"))
			if len(tr.columns)+n > maxTableColumns {
				tr.tooBig = true
				return skip(dec)
			}
			width := tr.size(styleRef{"table-column", attr(c, nsTable, "style-name")}, "style:column-width")
			for range n {
				tr.columns = append(tr.columns, width)
			}
			return skip(dec)
		case "table-row":
			return tr.readRow(dec, c)
		}
		return skip(dec)
	})
}

// readRow reads one table:table-row, and its repeats.
func (tr *tableReader) readRow(dec *officezip.Decoder, c xml.StartElement) error {
	n := repeated(attr(c, nsTable, "number-rows-repeated"))
	if len(tr.rows)+n > maxTableRows || (len(tr.rows)+n)*max(len(tr.columns), 1) > maxTableCells {
		tr.tooBig = true
		return skip(dec)
	}
	rowRef := styleRef{"table-row", attr(c, nsTable, "style-name")}
	height := tr.size(rowRef, "style:row-height")
	if height == 0 {
		height = tr.size(rowRef, "style:min-row-height")
	}
	defaultStyle := attr(c, nsTable, "default-cell-style-name")
	var row []outCell
	err := children(dec, func(cell xml.StartElement) error {
		covered := is(cell, nsTable, "covered-table-cell")
		if tr.tooBig || (!covered && !is(cell, nsTable, "table-cell")) {
			return skip(dec)
		}
		m := repeated(attr(cell, nsTable, "number-columns-repeated"))
		if len(row)+m > maxTableColumns || (len(tr.rows)+n)*(len(row)+m) > maxTableCells {
			tr.tooBig = true
			return skip(dec)
		}
		var out outCell
		if covered {
			if err := skip(dec); err != nil {
				return err
			}
		} else {
			var err error
			if out, err = tr.cell(dec, cell, defaultStyle); err != nil {
				return err
			}
		}
		for range m {
			row = append(row, out)
		}
		return nil
	})
	if err != nil || tr.tooBig {
		return err
	}
	for range n {
		tr.rows = append(tr.rows, height)
		tr.cells = append(tr.cells, append([]outCell(nil), row...))
	}
	return nil
}

// cell converts one table:table-cell: its spans, text, fill, borders and
// anchor. A cell that names no style takes its row's default.
func (tr *tableReader) cell(dec *officezip.Decoder, c xml.StartElement, defaultStyle string) (outCell, error) {
	s := tr.s
	style := attr(c, nsTable, "style-name")
	if style == "" {
		style = defaultStyle
	}
	r := []styleRef{{"table-cell", style}}
	var cell outCell
	if n, err := strconv.Atoi(attr(c, nsTable, "number-columns-spanned")); err == nil && n > 1 {
		cell.ColSpan = min(n, maxTableColumns)
	}
	if n, err := strconv.Atoi(attr(c, nsTable, "number-rows-spanned")); err == nil && n > 1 {
		cell.RowSpan = min(n, maxTableRows)
	}
	ctx := textContext{refs: append(r, tr.ctx.refs...)}
	paragraphs, err := s.textBody(dec, ctx)
	if err != nil {
		return outCell{}, err
	}
	// One empty line is a blank cell, which the editor writes as none.
	if hasText(paragraphs) || len(paragraphs) > 1 {
		cell.Paragraphs = paragraphs
	}
	if fill, _ := s.styles.prop(s.scope, r, "draw:fill"); fill == "solid" || fill == "gradient" || fill == "hatch" {
		cell.Fill = s.fillColor(r, fill)
	} else if fill == "" {
		bg, _ := s.styles.prop(s.scope, r, "fo:background-color")
		cell.Fill = parseColor(bg)
	}
	side := func(name string) *outStroke {
		v, ok := s.styles.prop(s.scope, r, "fo:border-"+name)
		if !ok {
			v, _ = s.styles.prop(s.scope, r, "fo:border")
		}
		return s.border(v)
	}
	borders := outBorders{Top: side("top"), Right: side("right"), Bottom: side("bottom"), Left: side("left")}
	if borders != (outBorders{}) {
		cell.Borders = &borders
	}
	anchor, ok := s.styles.prop(s.scope, r, "draw:textarea-vertical-align")
	if !ok {
		anchor, _ = s.styles.prop(s.scope, r, "style:vertical-align")
	}
	cell.Anchor = map[string]string{"middle": "middle", "bottom": "bottom"}[anchor]
	return cell, nil
}

// border reads an fo:border value, "0.5pt solid #000000", as a stroke; nil
// for none.
func (s *slideReader) border(v string) *outStroke {
	stroke := outStroke{Color: "#000000", Width: 1}
	drawn := false
	for _, part := range strings.Fields(v) {
		switch part {
		case "none", "hidden":
			return nil
		case "solid", "double", "groove", "ridge", "inset", "outset":
			drawn = true
		case "dashed":
			drawn, stroke.Dash = true, "dash"
		case "dotted":
			drawn, stroke.Dash = true, "dot"
		default:
			if c := parseColor(part); c != "" {
				stroke.Color = c
			} else if cm, ok := parseCm(part); ok && cm > 0 {
				stroke.Width = math.Max(s.units(cm), 0.01)
			}
		}
	}
	if !drawn {
		return nil
	}
	return &stroke
}

// size is a column's or row's size in slide units, 0 when its style sets
// none.
func (tr *tableReader) size(ref styleRef, key string) float64 {
	v, _ := tr.s.styles.prop(tr.s.scope, []styleRef{ref}, key)
	cm, ok := parseCm(v)
	if !ok || cm <= 0 {
		return 0
	}
	return tr.s.units(cm)
}

// repeated reads a number-*-repeated attribute: 1 when unset, and never
// more than one past the largest table, so a hostile count is refused
// rather than looped over.
func repeated(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	switch {
	case errors.Is(err, strconv.ErrRange) && !strings.HasPrefix(strings.TrimSpace(v), "-"):
		return maxTableCells + 1
	case err != nil || n < 1:
		return 1
	}
	return min(n, maxTableCells+1)
}
