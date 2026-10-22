package odputil

import (
	"fmt"
	"math"
	"strings"
)

// tableRow is one table:table-row, and how many times it repeats.
type tableRow struct {
	n      *node
	repeat int
}

// convertTable converts a frame's table:table, its columns and rows sized
// as their styles say, the frame's size shared among those that say
// nothing. A table past the editor's limits is skipped with a warning.
func (s *slideReader) convertTable(tbl *node, frame outFrame, frameRefs []styleRef) ([]outElement, error) {
	tooLarge := func() ([]outElement, error) {
		s.warn(fmt.Sprintf("Tables of more than %d rows, %d columns or %d cells are not imported.",
			maxTableRows, maxTableColumns, maxTableCells))
		return nil, nil
	}
	var columns []float64
	var rows []tableRow
	totalRows := 0
	var gather func(n *node) bool
	gather = func(n *node) bool {
		for _, c := range n.children {
			switch {
			case c.is(nsTable, "table-column"):
				repeat := max(atoiDefault(c.attr(nsTable, "number-columns-repeated"), 1), 1)
				if len(columns)+repeat > maxTableColumns {
					return false
				}
				ref := []styleRef{{scope: s.scope, family: "table-column", name: c.attr(nsTable, "style-name")}}
				width, ok := s.length(s.styles.get(ref, "table-column-properties", "style:column-width"))
				if !ok || width <= 0 {
					width = -1
				}
				for range repeat {
					columns = append(columns, width)
				}
			case c.is(nsTable, "table-row"):
				repeat := max(atoiDefault(c.attr(nsTable, "number-rows-repeated"), 1), 1)
				if repeat > maxTableRows || totalRows+repeat > maxTableRows {
					return false
				}
				totalRows += repeat
				rows = append(rows, tableRow{n: c, repeat: repeat})
			case c.is(nsTable, "table-columns"), c.is(nsTable, "table-header-columns"),
				c.is(nsTable, "table-rows"), c.is(nsTable, "table-header-rows"):
				if !gather(c) {
					return false
				}
			}
		}
		return true
	}
	if !gather(tbl) {
		return tooLarge()
	}
	// A row's cells, each with how many times it repeats; the table is as
	// wide as its widest row or its columns.
	width := len(columns)
	cellsOf := make([][]tableRow, len(rows))
	for i, row := range rows {
		count := 0
		for _, c := range row.n.children {
			if !c.is(nsTable, "table-cell") && !c.is(nsTable, "covered-table-cell") {
				continue
			}
			repeat := max(atoiDefault(c.attr(nsTable, "number-columns-repeated"), 1), 1)
			if repeat > maxTableColumns || count+repeat > maxTableColumns {
				return tooLarge()
			}
			count += repeat
			cellsOf[i] = append(cellsOf[i], tableRow{n: c, repeat: repeat})
		}
		width = max(width, count)
	}
	if totalRows == 0 || width == 0 {
		return nil, nil
	}
	if totalRows*width > maxTableCells {
		return tooLarge()
	}
	// Each cell is drawn as a shape is, so the deck's budget pays for it.
	if err := s.countElement(1 + totalRows*width); err != nil {
		return nil, err
	}

	out := &outTable{
		ID: s.id(), Type: typeTable, Frame: frame,
		Columns:    shareSizes(columns, width, frame.Width),
		HeaderRow:  tbl.attr(nsTable, "use-first-row-styles") == "true",
		BandedRows: tbl.attr(nsTable, "use-banding-rows-styles") == "true",
	}
	heights := make([]float64, 0, totalRows)
	for i, row := range rows {
		ref := []styleRef{{scope: s.scope, family: "table-row", name: row.n.attr(nsTable, "style-name")}}
		height, ok := s.length(s.styles.get(ref, "table-row-properties", "style:row-height"))
		if !ok {
			height, ok = s.length(s.styles.get(ref, "table-row-properties", "style:min-row-height"))
		}
		if !ok || height <= 0 {
			height = -1
		}
		cells := make([]outCell, width)
		col := 0
		for _, c := range cellsOf[i] {
			cell, err := s.tableCell(c.n, frameRefs)
			if err != nil {
				return nil, err
			}
			for range c.repeat {
				cells[col] = cell
				col++
			}
		}
		for range row.repeat {
			heights = append(heights, height)
			out.Cells = append(out.Cells, cells)
		}
	}
	out.Rows = shareSizes(heights, totalRows, frame.Height)
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

// shareSizes pads sizes to count and gives each unknown one, -1, an equal
// share of what total leaves. Every size is at least 1.
func shareSizes(sizes []float64, count int, total float64) []float64 {
	out := make([]float64, count)
	known, unknown := 0.0, 0
	for i := range out {
		out[i] = -1
		if i < len(sizes) {
			out[i] = sizes[i]
		}
		if out[i] < 0 {
			unknown++
		} else {
			known += out[i]
		}
	}
	share := 0.0
	if unknown > 0 {
		share = (total - known) / float64(unknown)
		if share <= 0 {
			share = total / float64(count)
		}
	}
	for i := range out {
		if out[i] < 0 {
			out[i] = share
		}
		out[i] = math.Max(round(out[i]), 1)
	}
	return out
}

// tableCell converts one cell. A covered cell keeps its text but no span;
// the editor finds it covered by its anchor's.
func (s *slideReader) tableCell(c *node, frameRefs []styleRef) (outCell, error) {
	ref := styleRef{scope: s.scope, family: "table-cell", name: c.attr(nsTable, "style-name")}
	refs := []styleRef{ref}
	paragraphs, err := s.paragraphs(c, append([]styleRef{ref}, frameRefs...))
	if err != nil {
		return outCell{}, err
	}
	var cell outCell
	if visibleText(paragraphs) {
		cell.Paragraphs = paragraphs
	}
	if c.is(nsTable, "table-cell") {
		if n := atoiDefault(c.attr(nsTable, "number-columns-spanned"), 1); n > 1 {
			cell.ColSpan = min(n, maxTableColumns)
		}
		if n := atoiDefault(c.attr(nsTable, "number-rows-spanned"), 1); n > 1 {
			cell.RowSpan = min(n, maxTableRows)
		}
	}
	if s.styles.get(refs, "graphic-properties", "draw:fill") == "solid" {
		cell.Fill = parseColor(s.styles.get(refs, "graphic-properties", "draw:fill-color"))
	}
	if cell.Fill == "" {
		cell.Fill = parseColor(s.styles.get(refs, "table-cell-properties", "fo:background-color"))
	}
	anchor := s.styles.get(refs, "graphic-properties", "draw:textarea-vertical-align")
	if anchor == "" {
		anchor = s.styles.get(refs, "table-cell-properties", "style:vertical-align")
	}
	cell.Anchor = map[string]string{"middle": "middle", "bottom": "bottom"}[anchor]
	side := func(name string) *outStroke {
		for _, kind := range []string{"table-cell-properties", "paragraph-properties"} {
			for _, key := range []string{"fo:border-" + name, "fo:border"} {
				if v, ok := s.styles.prop(refs, kind, key); ok {
					return s.border(v)
				}
			}
		}
		return nil
	}
	borders := outBorders{Top: side("top"), Right: side("right"), Bottom: side("bottom"), Left: side("left")}
	if borders != (outBorders{}) {
		cell.Borders = &borders
	}
	return cell, nil
}

// border reads an fo:border value, "0.05pt solid #000000", as a stroke, nil
// for none.
func (s *slideReader) border(v string) *outStroke {
	stroke := &outStroke{Color: "#000000", Width: 1}
	for _, field := range strings.Fields(v) {
		switch field {
		case "none", "hidden":
			return nil
		case "dashed":
			stroke.Dash = "dash"
		case "dotted":
			stroke.Dash = "dot"
		}
		if c := parseColor(field); c != "" {
			stroke.Color = c
		} else if w, ok := s.length(field); ok {
			if w <= 0 {
				return nil
			}
			stroke.Width = round(math.Max(w, 0.01))
		}
	}
	return stroke
}
