package docreadutil

// cspell:ignore aftncn aftnsep aftnsepc annotation ansicpg atnauthor atnid bkmkstart cellx chftn clmgf clmrg clvmgf clvmrg colortbl cpg deff emdash emspace endash enspace fcharset filetbl fldinst fldrslt fonttbl footerf footerl footerr fprq ftnalt ftncn ftnsep ftnsepc headerf headerl headerr ilvl intbl itap ldblquote listtext lquote ltrmark mmath nestcell nestrow nesttableprops nonesttables nonshppict nosupersub objdata outlinelevel pagebb pntext qmspace rdblquote revtbl rquote rtlmark rxe shppict shprslt striked stylesheet themedata trleft trowd twips txe uldash uldashd uldashdd uldb ulhwave ulldash ulnone ulth ulthd ulthdash ulthdashd ulthdashdd ulthldash ululdbwave ulw ulwave zwbo zwj zwnbo zwnj

import "strings"

// rtfWord is the meaning of a control word the RTF reader acts on.
type rtfWord uint8

// The control words the reader acts on. Every other one is the zero
// value, which is read and ignored.
const (
	_ rtfWord = iota

	// \* marks the destination that follows as one to skip unless it is
	// understood.
	wStar

	// Document properties.
	wRTF
	wAnsi
	wMac
	wPC
	wPCA
	wAnsiCpg
	wDeff
	wUC
	wU
	wBin

	// Destinations the reader reads.
	wFontTbl
	wStyleSheet
	wInfo
	wTitle
	wField
	wFldInst
	wFldRslt
	wListText
	wPnText
	wFootnote
	wShpPict
	wNestTableProps

	// Destinations the reader skips and says so.
	wPict
	wObject
	wShape
	wMath
	wHeaderFooter
	wAnnotation

	// Destinations the reader skips without a word: tables of fonts and
	// colors it does not need, revision and file tables, index and table of
	// contents entries, note separators and pictures for older readers.
	wSkipDest

	// Font table entries.
	wF
	wFCharset
	wCpg
	wFPrq

	// Style sheet entries.
	wS
	wCharStyle

	// Paragraph formatting.
	wPard
	wOutlineLevel
	wQL
	wQC
	wQR
	wQJ
	wInTbl
	wItap
	wIlvl
	wLs
	wPageBB

	// Character formatting.
	wPlain
	wB
	wI
	wUl
	wUlNone
	wStrike
	wStrikeD
	wSuper
	wSub
	wNoSuperSub
	wUp
	wDn
	wV
	wDeleted

	// Tables.
	wTrowd
	wTrLeft
	wCellX
	wClmgf
	wClmrg
	wClvmgf
	wClvmrg
	wCell
	wRow
	wNestCell
	wNestRow

	// Breaks and special characters.
	wPar
	wSect
	wPage
	wLine
	wTab
	wChftn

	// Special characters.
	wEmDash
	wEnDash
	wSpace
	wBullet
	wLQuote
	wRQuote
	wLDblQuote
	wRDblQuote
	wNBSpace
	wNBHyphen
)

// rtfWords maps the control words the reader acts on to their meaning.
var rtfWords = map[string]rtfWord{
	"rtf": wRTF, "ansi": wAnsi, "mac": wMac, "pc": wPC, "pca": wPCA, "ansicpg": wAnsiCpg, "deff": wDeff,
	"uc": wUC, "u": wU, "bin": wBin,

	"fonttbl": wFontTbl, "stylesheet": wStyleSheet, "info": wInfo, "title": wTitle,
	"field": wField, "fldinst": wFldInst, "fldrslt": wFldRslt,
	"listtext": wListText, "pntext": wPnText, "footnote": wFootnote,
	"shppict": wShpPict, "nesttableprops": wNestTableProps,

	"pict": wPict, "object": wObject, "shp": wShape, "do": wShape,
	"mmath": wMath, "moMath": wMath, "moMathPara": wMath,
	"header": wHeaderFooter, "headerl": wHeaderFooter, "headerr": wHeaderFooter, "headerf": wHeaderFooter,
	"footer": wHeaderFooter, "footerl": wHeaderFooter, "footerr": wHeaderFooter, "footerf": wHeaderFooter,
	"annotation": wAnnotation, "atnid": wAnnotation, "atnauthor": wAnnotation,

	"colortbl": wSkipDest, "filetbl": wSkipDest, "revtbl": wSkipDest, "nonshppict": wSkipDest,
	"shprslt": wSkipDest, "xe": wSkipDest, "tc": wSkipDest, "txe": wSkipDest, "rxe": wSkipDest,
	"ftnsep": wSkipDest, "ftnsepc": wSkipDest, "ftncn": wSkipDest,
	"aftnsep": wSkipDest, "aftnsepc": wSkipDest, "aftncn": wSkipDest, "private": wSkipDest,
	"objdata": wSkipDest, "nonesttables": wSkipDest, "datastore": wSkipDest, "themedata": wSkipDest,

	"f": wF, "fcharset": wFCharset, "cpg": wCpg, "fprq": wFPrq,

	"s": wS, "cs": wCharStyle, "ds": wCharStyle, "ts": wCharStyle,

	"pard": wPard, "outlinelevel": wOutlineLevel,
	"ql": wQL, "qc": wQC, "qr": wQR, "qj": wQJ, "qd": wQJ,
	"intbl": wInTbl, "itap": wItap, "ilvl": wIlvl, "ls": wLs, "pagebb": wPageBB,

	"plain": wPlain, "b": wB, "i": wI,
	"ul": wUl, "uld": wUl, "uldash": wUl, "uldashd": wUl, "uldashdd": wUl, "uldb": wUl,
	"ulhwave": wUl, "ulldash": wUl, "ulth": wUl, "ulthd": wUl, "ulthdash": wUl,
	"ulthdashd": wUl, "ulthdashdd": wUl, "ulthldash": wUl, "ululdbwave": wUl, "ulw": wUl,
	"ulwave": wUl, "ulnone": wUlNone,
	"strike": wStrike, "striked": wStrikeD, "super": wSuper, "sub": wSub,
	"nosupersub": wNoSuperSub, "up": wUp, "dn": wDn, "v": wV, "deleted": wDeleted,

	"trowd": wTrowd, "trleft": wTrLeft, "cellx": wCellX,
	"clmgf": wClmgf, "clmrg": wClmrg, "clvmgf": wClvmgf, "clvmrg": wClvmrg,
	"cell": wCell, "row": wRow, "nestcell": wNestCell, "nestrow": wNestRow,

	"par": wPar, "sect": wSect, "page": wPage, "line": wLine, "tab": wTab, "chftn": wChftn,

	"emdash": wEmDash, "endash": wEnDash, "emspace": wSpace, "enspace": wSpace, "qmspace": wSpace,
	"bullet": wBullet, "lquote": wLQuote, "rquote": wRQuote, "ldblquote": wLDblQuote, "rdblquote": wRDblQuote,
}

// rtfSymbols maps the control symbols the reader acts on to their meaning.
// The rest (\-, an optional hyphen, \: and \|) show nothing.
var rtfSymbols = [256]rtfWord{'*': wStar, '~': wNBSpace, '_': wNBHyphen}

// rtfCharacters are what the special-character control words and symbols
// stand for.
var rtfCharacters = map[rtfWord]rune{
	wEmDash: '\u2014', wEnDash: '\u2013', wSpace: ' ', wBullet: '\u2022',
	wLQuote: '\u2018', wRQuote: '\u2019', wLDblQuote: '\u201c', wRDblQuote: '\u201d',
	wNBSpace: '\u00a0', wNBHyphen: '-',
}

// rtfDest is where the text of a group goes.
type rtfDest uint8

const (
	// destBody is the document's text, or a note's.
	destBody rtfDest = iota
	// destSkip is a group whose content is read through and dropped.
	destSkip
	destFontTable
	destFontEntry
	destStyleSheet
	destStyleEntry
	destInfo
	destTitle
	destFieldInst
	destListText
	// destTableProps is a nested table row's properties, which come after
	// the row's cells; text there is dropped.
	destTableProps
)

// rtfCharProps is a group's character formatting.
type rtfCharProps struct {
	bold, italic, underline, strike, sup, sub bool
	// hidden is \v text, and deleted a tracked deletion; neither is shown.
	hidden, deleted bool
	// font is the \f font, or -1 for the document's default.
	font int
}

// rtfParaProps is a group's paragraph formatting.
type rtfParaProps struct {
	// outline is \outlinelevel, 0 to 9, or -1 when unset.
	outline int
	// style is the \s paragraph style.
	style int
	align string
	// intbl reports a paragraph in a table, and itap its nesting level,
	// -1 when unset.
	intbl bool
	itap  int
	// ilvl is a list item's level, and listed reports a \ls list.
	ilvl   int
	listed bool
	// pageBreakBefore is \pagebb.
	pageBreakBefore bool
}

// rtfState is what a group carries, and restores when it closes.
type rtfState struct {
	dest rtfDest
	char rtfCharProps
	para rtfParaProps
	// uc is how many fallback characters follow a \u.
	uc int
	// link is the hyperlink the group's text sits in.
	link string
	// field is the field the group sits in.
	field *rtfField
	// starred reports a \* not yet followed by its control word.
	starred bool
}

// rtfField is one \field: the instruction text read so far, kept to what
// identifies a hyperlink, and the link once its result has begun.
type rtfField struct {
	inst   strings.Builder
	link   string
	parsed bool
}

// rtfFont is one font table entry.
type rtfFont struct {
	// codePage is what the font's \fcharset or \cpg names, 0 for the
	// document's.
	codePage int
	// code reports a fixed-pitch font, whose text reads as code.
	code bool
}

// rtfFontEntry is the font table entry being read.
type rtfFontEntry struct {
	id   int
	font rtfFont
	name strings.Builder
}

// rtfStyleEntry is the style sheet entry being read.
type rtfStyleEntry struct {
	id      int
	outline int
	// paragraph is false for a character, section or table style.
	paragraph bool
	name      strings.Builder
}

// rtfParagraph is the paragraph being read: the DOCX reader's paragraph,
// with the list marker a \listtext gives it.
type rtfParagraph struct {
	paragraph
	marker    strings.Builder
	hasMarker bool
	// shaped reports that the block's type is settled: once part of the
	// paragraph has been placed, the rest is a plain paragraph.
	shaped bool
}

// rtfCellDef is a cell's definition in a row's properties.
type rtfCellDef struct {
	// right is the cell's right edge, \cellx, in twips.
	right int
	// hFirst and hNext are \clmgf and \clmrg, a horizontal merge; vFirst
	// and vNext are \clvmgf and \clvmrg, a vertical one.
	hFirst, hNext, vFirst, vNext bool
}

// rtfRowDefs is the properties of the rows at one table nesting level.
type rtfRowDefs struct {
	left  int
	cells []rtfCellDef
	// pending is the merge flags read for the cell whose \cellx is next.
	pending rtfCellDef
}

// rtfCell is a table cell as read, before its spans are worked out.
type rtfCell struct {
	blocks []Block
	// left and def place the cell on the table's grid; hasDef is false
	// for a cell the row's properties do not define.
	left   int
	def    rtfCellDef
	hasDef bool
}

// rtfTable is a table being read. RTF tables are flat: each row's
// paragraphs carry \intbl and \itap, \cell ends a cell and \row a row.
type rtfTable struct {
	rows [][]rtfCell
	// row is the cells of the row being read, and cur the blocks of the
	// cell being read.
	row []rtfCell
	cur []Block
}
