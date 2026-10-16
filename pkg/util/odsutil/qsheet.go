package odsutil

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// writeSheets writes the plans measured from an earlier read of the same
// part. The second walk exists because every row of a sheet is written at
// the sheet's full width, and that width is not known until the last row.
func writeSheets(ctx context.Context, dec *officezip.Decoder, plans []sheetPlan, out io.Writer) (ConvertToQsheetResult, error) {
	w := &sheetWriter{
		plans: plans,
		q:     &qsheetWriter{bw: bufio.NewWriter(out), remain: maxOutputBytes},
		ctx:   func() error { return canceled(ctx) },
	}
	if err := w.q.open(); err != nil {
		return w.q.result(), err
	}
	err := walkContent(dec, w.start, w.row)
	if err != nil {
		return w.q.result(), err
	}
	if err := w.finish(); err != nil {
		return w.q.result(), err
	}
	if w.index != len(plans) {
		return w.q.result(), fmt.Errorf("%w: the spreadsheet changed between reads", ErrMalformed)
	}
	if err := w.q.close(); err != nil {
		return w.q.result(), err
	}
	return w.q.result(), nil
}

func (w *sheetWriter) start(_ string) error {
	if err := w.ctx(); err != nil {
		return err
	}
	if err := w.finish(); err != nil {
		return err
	}
	if w.index >= len(w.plans) {
		return fmt.Errorf("%w: the spreadsheet changed between reads", ErrMalformed)
	}
	// The measure pass already chose the name, including the SheetN stand-in
	// for a table that did not name itself.
	plan := w.plans[w.index]
	nameCost := int64(len(plan.name))*6 + 64
	if nameCost > w.q.remain {
		return tooMuchOutput()
	}
	w.q.remain -= nameCost
	w.width = plan.width
	w.written = 0
	w.maxWidth = 0
	w.buf = make([]string, plan.width)
	w.blank = make([]string, plan.width)
	w.active = true
	return w.q.beginSheet(plan.name)
}

func (w *sheetWriter) row(repeat int64, huge bool, cells []cellRun) error {
	if err := w.ctx(); err != nil {
		return err
	}
	if !w.active {
		return fmt.Errorf("%w: a row outside a sheet", ErrMalformed)
	}
	blankN, copies, err := w.acc.take(repeat, huge, len(cells) > 0)
	if err != nil {
		return err
	}
	if blankN == 0 && copies == 0 {
		return nil
	}
	if len(cells) > 0 {
		width, err := runWidth(cells)
		if err != nil {
			return err
		}
		if width > w.maxWidth {
			w.maxWidth = width
		}
	}
	if err := w.pay(nil, blankN); err != nil {
		return err
	}
	if err := w.pay(cells, copies); err != nil {
		return err
	}
	for i := 0; i < blankN; i++ {
		if err := w.q.writeRow(w.blank); err != nil {
			return err
		}
	}
	if copies > 0 {
		if err := expandRow(cells, w.buf); err != nil {
			return err
		}
		for i := 0; i < copies; i++ {
			if err := w.q.writeRow(w.buf); err != nil {
				return err
			}
		}
	}
	w.written += blankN + copies
	return nil
}

func (w *sheetWriter) finish() error {
	if !w.active {
		return nil
	}
	w.acc.dropPending()
	plan := w.plans[w.index]
	if w.written != plan.rows || w.maxWidth != plan.width {
		return fmt.Errorf("%w: the spreadsheet changed between reads", ErrMalformed)
	}
	if w.written == 0 {
		// The editor treats a sheet with no rows as a broken document, so an
		// empty sheet is the one empty row that means the same thing.
		if err := w.q.writeRow([]string{""}); err != nil {
			return err
		}
	}
	w.index++
	w.active = false
	return w.q.endSheet()
}

// pay charges the JSON this row group will write, before any of it is written.
func (w *sheetWriter) pay(cells []cellRun, rows int) error {
	if rows <= 0 {
		return nil
	}
	text, ok := textCost(cells)
	if !ok {
		return tooMuchOutput()
	}
	more, ok := mulInt(text, int64(rows))
	if !ok {
		return tooMuchOutput()
	}
	return chargeOutput(&w.q.remain, more, rows, w.width)
}

func (q *qsheetWriter) open() error {
	_, err := q.bw.WriteString(`{"tabs":[`)
	return err
}

func (q *qsheetWriter) beginSheet(name string) error {
	if q.tabs > 0 {
		if err := q.bw.WriteByte(','); err != nil {
			return err
		}
	}
	if _, err := q.bw.WriteString(`{"name":`); err != nil {
		return err
	}
	if err := writeJSONString(q.bw, name); err != nil {
		return err
	}
	_, err := q.bw.WriteString(`,"data":{"rows":[`)
	q.needRow = false
	return err
}

func (q *qsheetWriter) writeRow(cells []string) error {
	if q.needRow {
		if err := q.bw.WriteByte(','); err != nil {
			return err
		}
	}
	q.needRow = true
	q.rows++
	for _, c := range cells {
		if c != "" {
			q.cells++
		}
	}
	return writeJSONRow(q.bw, cells)
}

func (q *qsheetWriter) endSheet() error {
	_, err := q.bw.WriteString(`]}}`)
	q.tabs++
	return err
}

func (q *qsheetWriter) close() error {
	if _, err := q.bw.WriteString(`]}`); err != nil {
		return err
	}
	return q.bw.Flush()
}

func (q *qsheetWriter) result() ConvertToQsheetResult {
	return ConvertToQsheetResult{Tabs: q.tabs, Rows: q.rows, Cells: q.cells}
}

// expandRow lays cells into buf, which has the sheet's width. Gaps in the
// middle stay empty and the tail past the last value is already empty.
func expandRow(cells []cellRun, buf []string) error {
	for i := range buf {
		buf[i] = ""
	}
	col := 0
	for _, c := range cells {
		if c.repeat > int64(len(buf)-col) {
			return tooWide()
		}
		n := int(c.repeat)
		if c.value != "" {
			for i := 0; i < n; i++ {
				buf[col+i] = c.value
			}
		}
		col += n
	}
	return nil
}

// writeJSONRow writes one row as a JSON array of strings.
func writeJSONRow(out *bufio.Writer, cells []string) error {
	if err := out.WriteByte('['); err != nil {
		return err
	}
	for i, cell := range cells {
		if i > 0 {
			if err := out.WriteByte(','); err != nil {
				return err
			}
		}
		if err := writeJSONString(out, cell); err != nil {
			return err
		}
	}
	return out.WriteByte(']')
}

// writeJSONString writes a quoted, escaped JSON string. Cell text is whatever
// the workbook held, so it is escaped by the JSON encoder rather than by hand.
func writeJSONString(out *bufio.Writer, s string) error {
	encoded, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = out.Write(encoded)
	return err
}
