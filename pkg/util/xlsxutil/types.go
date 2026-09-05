package xlsxutil

import (
	"archive/zip"
	"time"
)

// workbook is a resolved .xlsx package: the parts, indexed by name, and the
// worksheets in the order the workbook lists them.
type workbook struct {
	parts  partIndex
	sheets []sheetRef
	// epoch is the day serial 0 denotes. Almost every workbook counts from
	// 1899-12-30; one saved with the 1904 date system counts from 1904-01-01.
	epoch time.Time
}

// sheetRef is one worksheet: the tab name the user gave it, and the part
// holding its rows.
type sheetRef struct {
	name string
	part *zip.File
}

// partIndex maps a package-relative part name ("xl/worksheets/sheet1.xml") to
// its archive entry. Built once so a lookup does not walk the whole archive.
type partIndex map[string]*zip.File

// relationship is one entry of a .rels part: Target is resolved relative to
// the directory of the part the relationships belong to.
type relationship struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

// sharedItem is one entry of the shared-string table. Plain text sits in T;
// rich text is split across formatting runs in R, whose pieces concatenate
// back into the string the cell shows. Phonetic hints (rPh) nest their own
// <t> deeper and so match neither field, which is what we want.
type sharedItem struct {
	T *string `xml:"t"`
	R []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

// sheetRow is one <row> of a worksheet. Index is the row's 1-based position,
// which is declared rather than implied: a worksheet stores only the rows that
// hold something, so a gap in Index is a run of empty rows.
type sheetRow struct {
	Index int         `xml:"r,attr"`
	Cells []sheetCell `xml:"c"`
}

// sheetCell is one <c> of a row.
type sheetCell struct {
	// Ref is the cell's address ("B7"). Like a row's index it is declared, so
	// a gap between two cells is a run of empty ones.
	Ref string `xml:"r,attr"`
	// Type says how V is read: "s" indexes the shared-string table, "b" is a
	// boolean, "e" an error, "str" a formula's string result, "inlineStr" text
	// carried in IS, and an empty type is a number.
	Type string `xml:"t,attr"`
	// Style indexes the cell formats in styles.xml, which is the only place a
	// date says it is one.
	Style *int `xml:"s,attr"`
	// Value is the stored value, or a formula's cached result.
	Value *string `xml:"v"`
	// Inline is the text of an inlineStr cell, which carries it here instead
	// of in the shared table.
	Inline *sharedItem `xml:"is"`
}

// numberFormats says which cell styles render as dates, which is not something
// a cell records: a date is a number whose format code makes it one.
type numberFormats struct {
	// dateStyles maps a cell-format index to the kind of date it displays.
	dateStyles map[int]dateKind
}

// dateKind is the part of a date-formatted number that its format code shows.
type dateKind int

const (
	// notDate is a number displayed as a number.
	notDate dateKind = iota
	// dateOnly shows the day but not the time.
	dateOnly
	// timeOnly shows the time but not the day.
	timeOnly
	// dateAndTime shows both.
	dateAndTime
)
