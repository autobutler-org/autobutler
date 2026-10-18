package docreadutil

// cspell:ignore cellx clmgf clmrg clvmgf clvmrg intbl itap nestcell nestrow nesttableprops trleft trowd

import (
	"slices"
	"strings"
)

// Table bounds, beyond [Limits.MaxTableDepth].
const (
	// maxRTFCells bounds the cells one row's properties define; Word's
	// own limit is 63.
	maxRTFCells = 64
	// maxRTFGrid bounds the distinct cell edges one table's grid is
	// worked out from. Past it, every cell spans one column.
	maxRTFGrid = 4096
)

// rawLevel is the table nesting level a paragraph's formatting names,
// before the depth limit: 0 for the body.
func rawLevel(pp rtfParaProps) int {
	level := max(pp.itap, 0)
	if pp.intbl && level == 0 {
		level = 1
	}
	return level
}

// tableWord acts on a table control word, and reports whether it was one.
// Row properties are read in the body, and in a nested row's
// \nesttableprops, which follows the row's cells.
func (r *rtfReader) tableWord(st *rtfState, tok rtfToken) (bool, error) {
	switch tok.word {
	case wTrowd, wTrLeft, wCellX, wClmgf, wClmrg, wClvmgf, wClvmrg:
		if d := r.rowDefs(st); d != nil {
			d.word(tok)
		}
		return true, nil
	case wCell, wNestCell, wRow, wNestRow:
	default:
		return false, nil
	}
	if st.dest != destBody && (st.dest != destTableProps || tok.word != wNestRow) {
		return true, nil
	}
	level := 1
	if tok.word == wNestCell || tok.word == wNestRow {
		level = max(rawLevel(st.para), 2)
	}
	if tok.word == wCell || tok.word == wNestCell {
		return true, r.cellEnd(st, level)
	}
	return true, r.rowEnd(st, level)
}

// rowDefs is the row properties a table control word in this group sets,
// or nil past the depth limit.
func (r *rtfReader) rowDefs(st *rtfState) *rtfRowDefs {
	level := 1
	if st.dest == destTableProps {
		level = max(rawLevel(st.para), 2)
	}
	if level >= len(r.defs) {
		return nil
	}
	return &r.defs[level]
}

// word applies a row property.
func (d *rtfRowDefs) word(tok rtfToken) {
	switch tok.word {
	case wTrowd:
		*d = rtfRowDefs{cells: d.cells[:0]}
	case wTrLeft:
		d.left = tok.param
	case wClmgf:
		d.pending.hFirst = true
	case wClmrg:
		d.pending.hNext = true
	case wClvmgf:
		d.pending.vFirst = true
	case wClvmrg:
		d.pending.vNext = true
	case wCellX:
		if len(d.cells) < maxRTFCells {
			def := d.pending
			def.right = tok.param
			d.cells = append(d.cells, def)
		}
		d.pending = rtfCellDef{}
	}
}

// cellEnd ends a cell of the table at a nesting level: \cell at level 1,
// \nestcell deeper. Past the depth limit, and in a note, it only ends the
// paragraph.
func (r *rtfReader) cellEnd(st *rtfState, level int) error {
	if r.inNote || level > r.e.limits.MaxTableDepth {
		if r.p != nil {
			return r.endParagraph(r.level(st.para))
		}
		return nil
	}
	if r.p != nil {
		if err := r.endParagraph(level); err != nil {
			return err
		}
	}
	if err := r.toDepth(level); err != nil {
		return err
	}
	if err := r.e.charge(cellOverhead); err != nil {
		return err
	}
	t := r.tables[level-1]
	t.row = append(t.row, rtfCell{blocks: t.cur})
	t.cur = nil
	return nil
}

// rowEnd ends a row of the table at a nesting level: \row at level 1,
// \nestrow deeper.
func (r *rtfReader) rowEnd(st *rtfState, level int) error {
	if r.inNote || level > r.e.limits.MaxTableDepth {
		return nil
	}
	if r.p != nil {
		// Text after the row's last cell is a cell of its own.
		if err := r.cellEnd(st, level); err != nil {
			return err
		}
	}
	if len(r.tables) < level {
		return nil
	}
	if err := r.toDepth(level); err != nil {
		return err
	}
	if err := r.e.charge(rowOverhead); err != nil {
		return err
	}
	r.finishRow(r.tables[level-1], level)
	return nil
}

// finishRow ends a table's row, giving its cells the definitions the
// row's properties at that level hold.
func (r *rtfReader) finishRow(t *rtfTable, level int) {
	if len(t.cur) > 0 {
		t.row = append(t.row, rtfCell{blocks: t.cur})
		t.cur = nil
	}
	if len(t.row) == 0 {
		return
	}
	if level < len(r.defs) {
		d := &r.defs[level]
		left := d.left
		for i := range t.row {
			if i >= len(d.cells) {
				break
			}
			t.row[i].def, t.row[i].hasDef, t.row[i].left = d.cells[i], true, left
			left = d.cells[i].right
		}
	}
	t.rows = append(t.rows, t.row)
	t.row = nil
}

// toDepth closes and opens tables until level are open.
func (r *rtfReader) toDepth(level int) error {
	for len(r.tables) > level {
		if err := r.closeTable(); err != nil {
			return err
		}
	}
	for len(r.tables) < level {
		if err := r.e.admit(); err != nil {
			return err
		}
		if err := r.e.charge(blockOverhead); err != nil {
			return err
		}
		r.tables = append(r.tables, &rtfTable{})
	}
	return nil
}

// closeTable closes the innermost table and places it: into the cell it
// sits in, or out to the writer.
func (r *rtfReader) closeTable() error {
	n := len(r.tables)
	t := r.tables[n-1]
	r.finishRow(t, n)
	r.tables = r.tables[:n-1]
	b := Block{Type: BlockTable, Rows: buildRTFTable(t.rows)}
	if n > 1 {
		outer := r.tables[n-2]
		outer.cur = append(outer.cur, b)
		return nil
	}
	return r.e.write(b)
}

// buildRTFTable works out a table's spans. Every cell's edges go on one
// grid, so a cell spans the grid columns between its edges, which is how
// Word writes a merged cell; \clmrg and \clvmrg merge cells across and down
// as older writers do.
func buildRTFTable(read [][]rtfCell) []TableRow {
	var grid []int
	edges := map[int]bool{}
	for _, row := range read {
		for _, c := range row {
			if c.hasDef && c.def.right > c.left {
				edges[c.left], edges[c.def.right] = true, true
			}
		}
		if len(edges) > maxRTFGrid {
			edges = nil
			break
		}
	}
	for edge := range edges {
		grid = append(grid, edge)
	}
	slices.Sort(grid)

	rows := make([]TableRow, 0, len(read))
	// merging maps a grid column to the cell a vertical merge started
	// there, while the merge runs.
	merging := map[int]cellIndex{}
	for _, row := range read {
		out := TableRow{Cells: []TableCell{}}
		col := 0
		// into is the cell the previous cell of this row was merged into
		// from below, so an across-merged follower joins it too.
		var into *TableCell
		for _, c := range row {
			start, span := col, 1
			if grid != nil && c.hasDef && c.def.right > c.left {
				from, _ := slices.BinarySearch(grid, c.left)
				to, _ := slices.BinarySearch(grid, c.def.right)
				start = max(from, col)
				span = min(max(to-start, 1), maxSpan)
			}
			col = start + span
			if c.def.hNext && into != nil {
				into.Blocks = appendContent(into.Blocks, c.blocks)
				continue
			}
			if c.def.hNext && len(out.Cells) > 0 {
				prev := &out.Cells[len(out.Cells)-1]
				prev.ColSpan = min(prev.ColSpan+span, maxSpan)
				prev.Blocks = appendContent(prev.Blocks, c.blocks)
				continue
			}
			if origin, ok := merging[start]; ok && c.def.vNext {
				o := &rows[origin.row].Cells[origin.cell]
				o.RowSpan++
				o.Blocks = appendContent(o.Blocks, c.blocks)
				into = o
				continue
			}
			into = nil
			out.Cells = append(out.Cells, TableCell{ColSpan: span, RowSpan: 1, Blocks: nonNil(c.blocks)})
			if c.def.vFirst {
				merging[start] = cellIndex{row: len(rows), cell: len(out.Cells) - 1}
			} else {
				delete(merging, start)
			}
		}
		rows = append(rows, out)
	}
	return rows
}

// appendContent adds a merged-away cell's blocks to the cell it merged
// into, leaving out its empty paragraphs.
func appendContent(into, blocks []Block) []Block {
	for _, b := range blocks {
		if b.Type != BlockParagraph || len(b.Runs) > 0 {
			into = append(into, b)
		}
	}
	return into
}

// hyperlink is the target of a HYPERLINK field, when it is one a reader may
// open: an http, https or mailto link. A link to a bookmark in the
// document (\l) has none here.
func (f *rtfField) hyperlink() string {
	if !f.parsed {
		f.parsed, f.link = true, parseHyperlink(f.inst.String())
	}
	return f.link
}

// parseHyperlink reads a field's instructions as a HYPERLINK field.
func parseHyperlink(inst string) string {
	args := fieldArgs(inst)
	if len(args) == 0 || !strings.EqualFold(args[0], "HYPERLINK") {
		return ""
	}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case `\l`:
			return ""
		case `\o`, `\t`:
			// A tooltip or a target frame, then the link.
			i++
			continue
		}
		if strings.HasPrefix(args[i], `\`) {
			continue
		}
		return safeLink(args[i])
	}
	return ""
}

// fieldArgs splits a field's instructions into words, a quoted string
// being one word without its quotes.
func fieldArgs(inst string) []string {
	var args []string
	for {
		inst = strings.TrimLeft(inst, " \t\n")
		if inst == "" {
			return args
		}
		if inst[0] == '"' {
			end := strings.IndexByte(inst[1:], '"')
			if end < 0 {
				return append(args, inst[1:])
			}
			args = append(args, inst[1:end+1])
			inst = inst[end+2:]
			continue
		}
		end := strings.IndexAny(inst, " \t\n")
		if end < 0 {
			return append(args, inst)
		}
		args = append(args, inst[:end])
		inst = inst[end:]
	}
}
