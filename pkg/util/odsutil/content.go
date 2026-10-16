package odsutil

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
	"github.com/autobutler-org/quark/pkg/util/xlsxutil"
)

// convert measures the workbook and then writes it. Measure failing leaves
// Out untouched.
func convert(params ConvertToQsheetParams) (ConvertToQsheetResult, error) {
	pkg, err := officezip.Open(params.Source, params.Size, officezip.Limits{})
	if err != nil {
		return ConvertToQsheetResult{}, err
	}
	if err := checkPackage(pkg); err != nil {
		return ConvertToQsheetResult{}, err
	}
	var plans []sheetPlan
	err = readContent(pkg, func(dec *officezip.Decoder) error {
		var measureErr error
		plans, measureErr = measure(params.Ctx, dec)
		return measureErr
	})
	if err != nil {
		return ConvertToQsheetResult{}, err
	}
	var result ConvertToQsheetResult
	err = readContent(pkg, func(dec *officezip.Decoder) error {
		var writeErr error
		result, writeErr = writeSheets(params.Ctx, dec, plans, params.Out)
		return writeErr
	})
	return result, err
}

func readContent(pkg *officezip.Package, fn func(*officezip.Decoder) error) error {
	dec, err := pkg.XML(contentPart)
	if err != nil {
		return err
	}
	defer dec.Close()
	return fn(dec)
}

// measure walks content.xml once and records each sheet's name and used
// range. Repeats are accounted for here, so a repeat past a limit fails
// before any row is built.
func measure(ctx context.Context, dec *officezip.Decoder) ([]sheetPlan, error) {
	m := &measurer{ctx: func() error { return canceled(ctx) }, remain: maxOutputBytes}
	err := walkContent(dec, m.start, m.row)
	if err != nil {
		return nil, err
	}
	if err := m.finish(); err != nil {
		return nil, err
	}
	if len(m.plans) == 0 {
		return nil, fmt.Errorf("%w: content.xml has no spreadsheet", ErrMalformed)
	}
	return m.plans, nil
}

func (m *measurer) start(name string) error {
	if err := m.ctx(); err != nil {
		return err
	}
	if err := m.finish(); err != nil {
		return err
	}
	if len(m.plans) >= xlsxutil.MaxSheets {
		return tooManySheets()
	}
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Sheet%d", len(m.plans)+1)
	} else if len(name) > maxNameBytes {
		return fmt.Errorf("%w: a sheet name holds more than %d bytes", ErrTooLarge, maxNameBytes)
	}
	nameCost := int64(len(name))*6 + 64
	if nameCost > m.remain {
		return tooMuchOutput()
	}
	m.remain -= nameCost
	m.cur = &sheetPlan{name: name}
	m.sheetStart = m.acc.committed
	m.textBytes = 0
	return nil
}

func (m *measurer) row(repeat int64, huge bool, cells []cellRun) error {
	if err := m.ctx(); err != nil {
		return err
	}
	if m.cur == nil {
		return fmt.Errorf("%w: a row outside a sheet", ErrMalformed)
	}
	_, copies, err := m.acc.take(repeat, huge, len(cells) > 0)
	if err != nil {
		return err
	}
	if copies == 0 {
		return nil
	}
	w, err := runWidth(cells)
	if err != nil {
		return err
	}
	if w > m.cur.width {
		m.cur.width = w
	}
	cost, ok := textCost(cells)
	if !ok {
		return tooMuchOutput()
	}
	more, ok := mulInt(cost, int64(copies))
	if !ok {
		return tooMuchOutput()
	}
	sum, ok := addInt(m.textBytes, more)
	if !ok {
		return tooMuchOutput()
	}
	m.textBytes = sum
	return nil
}

func (m *measurer) finish() error {
	if m.cur == nil {
		return nil
	}
	m.acc.dropPending()
	m.cur.rows = m.acc.committed - m.sheetStart
	var err error
	m.grid, err = accountGrid(m.grid, m.cur.rows, m.cur.width)
	if err != nil {
		return err
	}
	if err := chargeOutput(&m.remain, m.textBytes, m.cur.rows, m.cur.width); err != nil {
		return err
	}
	m.plans = append(m.plans, *m.cur)
	m.cur = nil
	return nil
}

// walkContent reports every sheet table and every row inside it. Elements
// that are not the spreadsheet's tables — named ranges, scripts, shapes —
// are skipped whole, so a table nested in one of them is not a sheet.
func walkContent(dec *officezip.Decoder, onSheet func(string) error, onRow func(int64, bool, []cellRun) error) error {
	tables := 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "spreadsheet" {
			continue
		}
		n, err := walkSpreadsheet(dec, onSheet, onRow)
		tables += n
		if err != nil {
			return err
		}
	}
	if tables == 0 {
		return fmt.Errorf("%w: content.xml has no spreadsheet", ErrMalformed)
	}
	return nil
}

func walkSpreadsheet(dec *officezip.Decoder, onSheet func(string) error, onRow func(int64, bool, []cellRun) error) (int, error) {
	depth := 1
	tables := 0
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return tables, fmt.Errorf("%w: content.xml ended early", ErrMalformed)
			}
			return tables, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "table" {
				tables++
				if err := walkTable(dec, t, onSheet, onRow); err != nil {
					return tables, err
				}
				continue
			}
			if err := dec.Skip(); err != nil {
				return tables, err
			}
		case xml.EndElement:
			depth--
		}
	}
	return tables, nil
}

func walkTable(dec *officezip.Decoder, start xml.StartElement, onSheet func(string) error, onRow func(int64, bool, []cellRun) error) error {
	if err := onSheet(attrFirst(start, "name")); err != nil {
		return err
	}
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("%w: content.xml ended early", ErrMalformed)
			}
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "table-row":
				if err := walkRow(dec, t, onRow); err != nil {
					return err
				}
			case "table-row-group", "table-header-rows", "table-rows":
				depth++
			default:
				// Columns, shapes and the rest declare no cells. Skipping
				// them also skips any table they happen to contain.
				if err := dec.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			depth--
		}
	}
	return nil
}

func walkRow(dec *officezip.Decoder, start xml.StartElement, onRow func(int64, bool, []cellRun) error) error {
	repeat, huge, err := repeatAttr(start, "number-rows-repeated")
	if err != nil {
		return err
	}
	var b rowBuilder
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("%w: content.xml ended early", ErrMalformed)
			}
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "table-cell", "covered-table-cell":
				value, err := readCell(dec, t)
				if err != nil {
					return err
				}
				colRepeat, colHuge, err := repeatAttr(t, "number-columns-repeated")
				if err != nil {
					return err
				}
				if err := b.add(colRepeat, colHuge, value); err != nil {
					return err
				}
			default:
				if err := dec.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			depth--
		}
	}
	// b.cells is reused only for this call; onRow must not keep it.
	return onRow(repeat, huge, b.cells)
}

// add records one cell. Empty cells pool into a pending run that is thrown
// away when the row ends, which is how a sheet's trailing empty columns
// disappear without being expanded. A pending run past MaxColumns becomes
// an error only when a later cell has a value.
func (b *rowBuilder) add(repeat int64, huge bool, value string) error {
	if value == "" {
		if huge || b.pendingHuge || repeat > int64(xlsxutil.MaxColumns) || b.pending > int64(xlsxutil.MaxColumns)-repeat {
			b.pendingHuge = true
			return nil
		}
		b.pending += repeat
		return nil
	}
	if huge || b.pendingHuge || repeat > int64(xlsxutil.MaxColumns) {
		return tooWide()
	}
	used := b.width()
	if b.pending > 0 {
		if used > int64(xlsxutil.MaxColumns)-b.pending || used+b.pending > int64(xlsxutil.MaxColumns)-repeat {
			return tooWide()
		}
		b.cells = append(b.cells, cellRun{repeat: b.pending})
		b.pending = 0
	} else if used > int64(xlsxutil.MaxColumns)-repeat {
		return tooWide()
	}
	b.cells = append(b.cells, cellRun{repeat: repeat, value: value})
	return nil
}

func (b *rowBuilder) width() int64 {
	var n int64
	for _, c := range b.cells {
		n += c.repeat
	}
	return n
}

// take records one row group. blank and copies are the rows the caller
// writes; an empty group only extends the pending hole and returns zeros.
// The hole is dropped by dropPending at the end of the sheet, so the empty
// rows a spreadsheet repeats out to its edge cost nothing.
func (a *rowAccount) take(repeat int64, huge, hasContent bool) (blank, copies int, err error) {
	if !hasContent {
		if huge || a.pendingHuge || repeat > int64(xlsxutil.MaxRows) || a.pending > int64(xlsxutil.MaxRows)-repeat {
			a.pendingHuge = true
			return 0, 0, nil
		}
		a.pending += repeat
		return 0, 0, nil
	}
	if huge || a.pendingHuge || repeat > int64(xlsxutil.MaxRows) {
		return 0, 0, tooManyRows()
	}
	used := int64(a.committed) + a.pending
	if a.pending > int64(xlsxutil.MaxRows) || used > int64(xlsxutil.MaxRows)-repeat {
		return 0, 0, tooManyRows()
	}
	blank = int(a.pending)
	copies = int(repeat)
	a.committed += blank + copies
	a.pending = 0
	return blank, copies, nil
}

func (a *rowAccount) dropPending() {
	a.pending = 0
	a.pendingHuge = false
}

// repeatAttr reads a repeat count. An absent attribute means one. A number
// that does not fit in an int64 is huge, which the caller drops when the
// run is a trailing gap and refuses when it is not.
func repeatAttr(start xml.StartElement, local string) (int64, bool, error) {
	raw := strings.TrimSpace(attrFirst(start, local))
	if raw == "" {
		return 1, false, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		var ne *strconv.NumError
		if errors.As(err, &ne) && errors.Is(ne.Err, strconv.ErrRange) {
			return 0, true, nil
		}
		return 0, false, fmt.Errorf("%w: repeat %q is not a positive integer", ErrMalformed, raw)
	}
	if n < 1 {
		return 0, false, fmt.Errorf("%w: repeat %q is not a positive integer", ErrMalformed, raw)
	}
	return n, false, nil
}

func readCell(dec *officezip.Decoder, start xml.StartElement) (string, error) {
	text, err := readCellText(dec)
	if err != nil {
		return "", err
	}
	// A covered cell is the tail of a merge. Its value lives in the origin
	// cell, so the tail is empty whatever it contains.
	if start.Name.Local != "table-cell" {
		return "", nil
	}
	return cellValue(start, text)
}

// readCellText reads the body of the cell start already consumed, through
// its end tag. Paragraphs join with newlines. Comments and embedded objects
// are skipped, so their text does not become the value.
func readCellText(dec *officezip.Decoder) (string, error) {
	var b strings.Builder
	depth := 1
	inPara := false
	paras := 0
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", fmt.Errorf("%w: content.xml ended early", ErrMalformed)
			}
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if skipInCell(t.Name.Local) {
				if err := dec.Skip(); err != nil {
					return "", err
				}
				continue
			}
			switch t.Name.Local {
			case "p", "h":
				if paras > 0 {
					if err := appendText(&b, "\n"); err != nil {
						return "", err
					}
				}
				paras++
				inPara = true
			case "s":
				if inPara {
					n, err := spaceCount(t)
					if err != nil {
						return "", err
					}
					if err := appendText(&b, strings.Repeat(" ", n)); err != nil {
						return "", err
					}
				}
			case "tab":
				if inPara {
					if err := appendText(&b, "\t"); err != nil {
						return "", err
					}
				}
			case "line-break":
				if inPara {
					if err := appendText(&b, "\n"); err != nil {
						return "", err
					}
				}
			}
			depth++
		case xml.EndElement:
			if t.Name.Local == "p" || t.Name.Local == "h" {
				inPara = false
			}
			depth--
		case xml.CharData:
			if inPara {
				if err := appendBytes(&b, t); err != nil {
					return "", err
				}
			}
		}
	}
	return b.String(), nil
}

func skipInCell(local string) bool {
	switch local {
	case "annotation", "annotation-end", "frame", "custom-shape", "image", "object", "object-ole":
		return true
	default:
		return false
	}
}

func spaceCount(start xml.StartElement) (int, error) {
	raw := strings.TrimSpace(attrFirst(start, "c"))
	if raw == "" {
		return 1, nil
	}
	n, huge, err := repeatAttr(start, "c")
	if err != nil {
		return 0, err
	}
	if huge || n > maxSpaceRun {
		return 0, fmt.Errorf("%w: a cell repeats spaces more than %d times", ErrTooLarge, maxSpaceRun)
	}
	return int(n), nil
}

func appendText(b *strings.Builder, s string) error {
	return appendBytes(b, []byte(s))
}

func appendBytes(b *strings.Builder, p []byte) error {
	if len(p) > maxCellBytes || b.Len() > maxCellBytes-len(p) {
		return tooLongCell()
	}
	_, err := b.Write(p)
	return err
}

func runWidth(cells []cellRun) (int, error) {
	var n int64
	for _, c := range cells {
		if c.repeat <= 0 || n > int64(xlsxutil.MaxColumns)-c.repeat {
			return 0, tooWide()
		}
		n += c.repeat
	}
	return int(n), nil
}

func accountGrid(total, rows, width int) (int, error) {
	if rows <= 0 || width <= 0 {
		return total, nil
	}
	if rows > xlsxutil.MaxRows || width > xlsxutil.MaxColumns {
		return 0, tooManyCells()
	}
	prod := int64(rows) * int64(width)
	if int64(total) > int64(xlsxutil.MaxCells) || prod > int64(xlsxutil.MaxCells)-int64(total) {
		return 0, tooManyCells()
	}
	return total + int(prod), nil
}

// textCost is a high estimate of the JSON bytes of a row's values, before
// the row is repeated. Escaping can turn one byte into \u00XX, six bytes.
func textCost(cells []cellRun) (int64, bool) {
	var n int64
	for _, c := range cells {
		if c.value == "" || c.repeat <= 0 {
			continue
		}
		per := int64(len(c.value))*6 + 2
		part, ok := mulInt(per, c.repeat)
		if !ok {
			return 0, false
		}
		n, ok = addInt(n, part)
		if !ok {
			return 0, false
		}
	}
	return n, true
}

// chargeOutput subtracts the grid's JSON framing and its text from remain.
func chargeOutput(remain *int64, textBytes int64, rows, width int) error {
	if rows <= 0 {
		return nil
	}
	frame, ok := mulInt(int64(rows), int64(width)*3+2)
	if !ok || textBytes > *remain || frame > *remain-textBytes {
		return tooMuchOutput()
	}
	*remain -= textBytes + frame
	return nil
}
