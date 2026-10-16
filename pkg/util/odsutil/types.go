package odsutil

import "bufio"

// Bounds that are this package's own. The sheet's shape is bounded by
// xlsxutil; these bound what a repeat does to memory and to the bytes
// written, which the cell count does not.
const (
	// maxCellBytes bounds one cell's text. A longer value is refused.
	maxCellBytes = 1 << 20 // 1 MiB
	// maxNameBytes bounds one sheet's name.
	maxNameBytes = 1024
	// maxOutputBytes bounds the JSON envelope. Repeating one long cell
	// across a legal grid would otherwise write far more than the part
	// that held it.
	maxOutputBytes = 256 << 20 // 256 MiB
	// maxSpaceRun bounds text:c, which expands to spaces that are not in
	// the file. A larger run is refused before it is allocated.
	maxSpaceRun = 1_000_000

	nsOffice    = "urn:oasis:names:tc:opendocument:xmlns:office:1.0"
	contentPart = "content.xml"
	mimePart    = "mimetype"

	mimeSpreadsheet = "application/vnd.oasis.opendocument.spreadsheet"
	mimeTemplate    = "application/vnd.oasis.opendocument.spreadsheet-template"
)

// sheetPlan is one sheet after the measure pass: the name to write, and how
// many rows and columns hold anything. Width is the column of the last
// value, so trailing empty repeats are already gone.
type sheetPlan struct {
	name  string
	rows  int
	width int
}

// cellRun is a contiguous run of cells that hold the same value. An empty
// value is a gap. Runs are what the file stored; they are expanded only
// while a row is written.
type cellRun struct {
	repeat int64
	value  string
}

// rowBuilder accumulates one row's cells up to its last value. Empty cells
// wait in pending so a trailing run can be dropped instead of stored.
type rowBuilder struct {
	cells       []cellRun
	pending     int64
	pendingHuge bool
}

// rowAccount counts rows across the workbook. Empty rows wait in pending
// until a later row has a value, and are dropped at the end of a sheet.
type rowAccount struct {
	committed   int
	pending     int64
	pendingHuge bool
}

// qsheetWriter streams the envelope. remain is how many bytes may still be
// written; a row group that would pass it is refused before it is written.
type qsheetWriter struct {
	bw      *bufio.Writer
	tabs    int
	rows    int
	cells   int
	remain  int64
	needRow bool
}

// measurer is the state of the measure pass.
type measurer struct {
	ctx        func() error
	plans      []sheetPlan
	acc        rowAccount
	cur        *sheetPlan
	sheetStart int
	textBytes  int64
	grid       int
	remain     int64
}

// sheetWriter is the state of the write pass.
type sheetWriter struct {
	plans    []sheetPlan
	index    int
	acc      rowAccount
	q        *qsheetWriter
	ctx      func() error
	width    int
	written  int
	maxWidth int
	buf      []string
	blank    []string
	active   bool
}
