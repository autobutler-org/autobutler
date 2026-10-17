package docreadutil

import (
	"encoding/json"
	"io"
	"strings"
)

// MarshalJSON writes a block with exactly the fields its type uses: a text
// block always has its runs, a list item its ordering, level and marker, a
// table its rows and every cell its spans and blocks.
func (b Block) MarshalJSON() ([]byte, error) {
	switch b.Type {
	case BlockHeading:
		return json.Marshal(headingJSON{Type: b.Type, Level: b.Level, Runs: nonNil(b.Runs)})
	case BlockParagraph:
		return json.Marshal(paragraphJSON{Type: b.Type, Align: b.Align, Runs: nonNil(b.Runs)})
	case BlockListItem:
		return json.Marshal(listItemJSON{Type: b.Type, Ordered: b.Ordered, Level: b.Level, Marker: b.Marker, Align: b.Align, Runs: nonNil(b.Runs)})
	case BlockTable:
		rows := nonNil(b.Rows)
		for i := range rows {
			rows[i].Cells = nonNil(rows[i].Cells)
			for j := range rows[i].Cells {
				rows[i].Cells[j].Blocks = nonNil(rows[i].Cells[j].Blocks)
			}
		}
		return json.Marshal(tableJSON{Type: b.Type, Rows: rows})
	case BlockImage:
		return json.Marshal(imageJSON{Type: b.Type, Part: b.Part, External: b.External, Alt: b.Alt, Width: b.Width, Height: b.Height})
	case BlockOmitted:
		return json.Marshal(omittedJSON{Type: b.Type, What: b.What})
	}
	return json.Marshal(typeJSON{Type: b.Type})
}

// nonNil returns s, or an empty slice for nil, so it marshals as [].
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// newNDJSONWriter returns a writer of the viewer's stream to w.
func newNDJSONWriter(w io.Writer) *ndjsonWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &ndjsonWriter{enc: enc}
}

func (n *ndjsonWriter) begin(meta metaLine) error { return n.enc.Encode(meta) }
func (n *ndjsonWriter) block(b Block) error       { return n.enc.Encode(b) }
func (n *ndjsonWriter) end(end endLine) error     { return n.enc.Encode(end) }
func (n *ndjsonWriter) losses() []string          { return nil }

func (d *deltaWriter) begin(metaLine) error {
	_, err := io.WriteString(d.w, `{"ops":[`)
	return err
}

// block writes a block as Delta ops: its runs, then the newline that
// carries its line format. A table is flattened to a line per row; an
// image, an omitted block and a page break have no Delta form.
func (d *deltaWriter) block(b Block) error {
	switch b.Type {
	case BlockHeading:
		return d.line(b.Runs, &deltaAttrs{Header: min(max(b.Level, 1), 6)})
	case BlockParagraph:
		return d.line(b.Runs, alignAttrs(b.Align))
	case BlockListItem:
		list := "bullet"
		if b.Ordered {
			list = "ordered"
		}
		return d.line(b.Runs, &deltaAttrs{List: list, Indent: b.Level, Align: b.Align})
	case BlockTable:
		d.tables++
		for _, row := range b.Rows {
			var runs []Run
			for i, cell := range row.Cells {
				if i > 0 {
					runs = append(runs, Run{Text: "\t"})
				}
				runs = d.cellRuns(runs, cell.Blocks)
			}
			if err := d.line(runs, nil); err != nil {
				return err
			}
		}
	case BlockImage:
		d.images++
	}
	return nil
}

// cellRuns appends the text of a table cell's blocks, flattened onto one
// line: blocks are separated by a space, and line breaks become spaces.
func (d *deltaWriter) cellRuns(runs []Run, blocks []Block) []Run {
	first := true
	for _, b := range blocks {
		switch b.Type {
		case BlockImage:
			d.images++
			continue
		case BlockTable:
			d.tables++
			for _, row := range b.Rows {
				for _, cell := range row.Cells {
					runs = d.cellRuns(runs, cell.Blocks)
				}
			}
			continue
		}
		if len(b.Runs) == 0 {
			continue
		}
		if !first {
			runs = append(runs, Run{Text: " "})
		}
		first = false
		for _, r := range b.Runs {
			r.Text = strings.ReplaceAll(r.Text, "\n", " ")
			runs = append(runs, r)
		}
	}
	return runs
}

// line writes one line: its runs, with every line break in them taking the
// line's format, and the newline that ends it.
func (d *deltaWriter) line(runs []Run, lineAttrs *deltaAttrs) error {
	for _, r := range runs {
		for i, piece := range strings.Split(r.Text, "\n") {
			if i > 0 {
				if err := d.op(deltaOp{Insert: "\n", Attributes: lineAttrs}); err != nil {
					return err
				}
			}
			if piece == "" {
				continue
			}
			if err := d.op(deltaOp{Insert: piece, Attributes: inlineAttrs(r)}); err != nil {
				return err
			}
		}
	}
	return d.op(deltaOp{Insert: "\n", Attributes: lineAttrs})
}

// op writes one op, after a comma when it is not the first.
func (d *deltaWriter) op(op deltaOp) error {
	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	if d.ops > 0 {
		data = append([]byte{','}, data...)
	}
	d.ops++
	_, err = d.w.Write(data)
	return err
}

func (d *deltaWriter) end(endLine) error {
	// A Delta document ends with a newline; an empty one is just that.
	if d.ops == 0 {
		if err := d.op(deltaOp{Insert: "\n"}); err != nil {
			return err
		}
	}
	_, err := io.WriteString(d.w, "]}")
	return err
}

func (d *deltaWriter) losses() []string {
	var out []string
	if d.tables > 0 {
		out = append(out, countPhrase(d.tables, "table", "tables")+" flattened to text")
	}
	if d.images > 0 {
		out = append(out, countPhrase(d.images, "image", "images")+" left out")
	}
	return out
}

// inlineAttrs is a run's formatting as Delta attributes, nil when it has
// none.
func inlineAttrs(r Run) *deltaAttrs {
	a := deltaAttrs{Bold: r.Bold, Italic: r.Italic, Underline: r.Underline, Strike: r.Strike, Code: r.Code, Link: r.Link}
	switch {
	case r.Superscript:
		a.Script = "super"
	case r.Subscript:
		a.Script = "sub"
	}
	if a == (deltaAttrs{}) {
		return nil
	}
	return &a
}

// alignAttrs is a paragraph's alignment as a Delta line format.
func alignAttrs(align string) *deltaAttrs {
	if align == "" {
		return nil
	}
	return &deltaAttrs{Align: align}
}
