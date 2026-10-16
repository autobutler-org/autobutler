// Package odsutil reads an OpenDocument spreadsheet and writes it as the
// .qsheet JSON envelope Quark's Sheets editor already understands:
//
//	{"tabs":[{"name":"Sheet1","data":{"rows":[["a","b"], …]}}]}
//
// Converting on read, rather than teaching Sheets a second format, is the
// same trade [xlsxutil.ConvertToQsheet] makes. The package opens the file
// through officezip: the caller supplies an [io.ReaderAt] and its size,
// content.xml is decoded as a token stream, and rows are written to Out as
// they are read. A sheet is never held. The editor takes a sheet's column
// count from its first row, so the width is measured in a first pass that
// keeps only that width, and the rows are written on the second.
//
// Cell values come from office:value-type and the matching value attribute,
// and from the cell's text when the attribute is absent. A formula
// contributes the cached value stored beside it, never the formula. Dates
// and times are written the way xlsxutil writes them, booleans as TRUE and
// FALSE.
//
// table:number-rows-repeated and table:number-columns-repeated are counted
// against xlsxutil's sheet limits before a single repeated row is written.
// A trailing run of empty repeats — what a spreadsheet writes out to the
// sheet's edge — is dropped and not expanded. Expanding one cell's text
// across those repeats is bounded again by an output cap, because the cell
// limits count cells and a long string repeated across a legal number of
// them would otherwise be an output bomb.
//
// Nothing in the package is executed or fetched. Macros, scripts and
// objects are never opened. An encrypted content.xml is [ErrEncrypted].
package odsutil

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/autobutler-org/quark/pkg/util/officezip"
	"github.com/autobutler-org/quark/pkg/util/xlsxutil"
)

// ErrMalformed is a file that is not a well-formed OpenDocument spreadsheet:
// not a zip, a zip with no spreadsheet, or a part that does not parse. It is
// [officezip.ErrMalformed], so either name matches.
var ErrMalformed = officezip.ErrMalformed

// ErrTooLarge is a workbook past a limit: the package's own, xlsxutil's sheet
// limits, or the output cap. It is [officezip.ErrTooLarge], so either name
// matches. Reaching a limit refuses the workbook; it is never truncated.
var ErrTooLarge = officezip.ErrTooLarge

// ErrEncrypted is a workbook whose content.xml is password-protected. The
// bytes are not decrypted and not parsed.
var ErrEncrypted = errors.New("odsutil: the workbook is encrypted")

// ConvertToQsheetParams is one spreadsheet on its way to becoming a .qsheet.
type ConvertToQsheetParams struct {
	// Ctx ends the conversion when it is done. Nil never does. A canceled
	// context returns that error and stops reading.
	Ctx context.Context
	// Source is the .ods package. A zip is read from its trailing central
	// directory, so this is a ReaderAt rather than a stream.
	Source io.ReaderAt
	// Size is Source's length in bytes, from the source's own Stat.
	Size int64
	// Out receives the .qsheet JSON. Nothing is written until the workbook
	// has been measured, so a file that is not a spreadsheet, or one past a
	// limit, fails before the first byte.
	Out io.Writer
}

// ConvertToQsheetResult reports what the converted workbook came to.
type ConvertToQsheetResult struct {
	// Tabs is the number of sheets written.
	Tabs int
	// Rows is the number of rows written across every sheet.
	Rows int
	// Cells is the number of cells written that hold a value.
	Cells int
}

// ConvertToQsheet reads the spreadsheet in params.Source and writes the
// equivalent .qsheet envelope to params.Out.
//
// Errors wrap [ErrMalformed] when the file is not a spreadsheet,
// [ErrTooLarge] when it passes a limit, and [ErrEncrypted] when content.xml
// is password-protected. A canceled params.Ctx returns that context's error.
// Anything else is a read or write failure. Out is untouched unless the
// write itself fails.
func ConvertToQsheet(params ConvertToQsheetParams) (ConvertToQsheetResult, error) {
	if params.Source == nil || params.Out == nil {
		return ConvertToQsheetResult{}, errors.New("odsutil: Source and Out are required")
	}
	if params.Size <= 0 {
		return ConvertToQsheetResult{}, fmt.Errorf("%w: empty file", ErrMalformed)
	}
	if err := canceled(params.Ctx); err != nil {
		return ConvertToQsheetResult{}, err
	}
	result, err := convert(params)
	if err != nil {
		return result, classify(err)
	}
	return result, nil
}

// Sheet limits are xlsxutil's. They are named here so a reader of this
// package sees the bounds the repeats are counted against.
const (
	// MaxSheets is the number of sheets read from one workbook.
	MaxSheets = xlsxutil.MaxSheets
	// MaxRows is the number of rows read from one workbook, every sheet
	// together.
	MaxRows = xlsxutil.MaxRows
	// MaxColumns is the number of columns read from one sheet.
	MaxColumns = xlsxutil.MaxColumns
	// MaxCells is the number of cells in the rectangular grid of the
	// workbook, empty padding included.
	MaxCells = xlsxutil.MaxCells
)
