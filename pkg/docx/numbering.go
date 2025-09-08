package docx

import (
	"fmt"

	"autobutler/pkg/docx/types"
	"github.com/nbio/xml"
)

// This element specifies the contents of a main document part in a WordprocessingML document.
type Numbering struct {
	XMLName      xml.Name       `xml:"w:numbering"`
	XMLNSa       string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:a,attr"`
	XMLNSc       string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:c,attr"`
	XMLNScr      string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:cr,attr"`
	XMLNSdgm     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:dgm,attr"`
	XMLNSlc      string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:lc,attr"`
	XMLNSm       string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:m,attr"`
	XMLNSmc      string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:mc,attr"`
	XMLNSo       string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:o,attr"`
	XMLNSpic     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:pic,attr"`
	XMLNSr       string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:r,attr"`
	XMLNSsl      string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:sl,attr"`
	XMLNSv       string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:v,attr"`
	XMLNSw       string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:w,attr"`
	XMLNSw10     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:w10,attr"`
	XMLNSw14     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:w14,attr"`
	XMLNSw15     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:w15,attr"`
	XMLNSw16     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:w16,attr"`
	XMLNSw16cex  string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:w16cex,attr"`
	XMLNSw16cid  string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:w16cid,attr"`
	XMLNSwne     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:wne,attr"`
	XMLNSwp      string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:wp,attr"`
	XMLNSwpg     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:wpg,attr"`
	XMLNSwps     string         `xml:"http://www.w3.org/2000/xmlns/ xmlns:wps,attr"`
	Namespace    string         `xml:"xmlns,attr"`
	AbstractNums []*AbstractNum `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:abstractNum"`
	Nums         []*Num         `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:num"`
	RelativePath string         `xml:"-"` // RelativePath is the path to the numbering file within the document package.
}

func NewNumbering() *Numbering {
	return &Numbering{
		// XMLName:          xml.Name{Local: "numbering"},
		AbstractNums: []*AbstractNum{},
		Nums:         []*Num{},
		XMLNSmc:      "http://schemas.openxmlformats.org/markup-compatibility/2006",
		XMLNSo:       "urn:schemas-microsoft-com:office:office",
		XMLNSr:       "http://schemas.openxmlformats.org/officeDocument/2006/relationships",
		XMLNSm:       "http://schemas.openxmlformats.org/officeDocument/2006/math",
		XMLNSv:       "urn:schemas-microsoft-com:vml",
		XMLNSwp:      "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing",
		XMLNSw10:     "urn:schemas-microsoft-com:office:word",
		XMLNSw:       "http://schemas.openxmlformats.org/wordprocessingml/2006/main",
		XMLNSwne:     "http://schemas.microsoft.com/office/word/2006/wordml",
		XMLNSsl:      "http://schemas.openxmlformats.org/schemaLibrary/2006/main",
		XMLNSa:       "http://schemas.openxmlformats.org/drawingml/2006/main",
		XMLNSpic:     "http://schemas.openxmlformats.org/drawingml/2006/picture",
		XMLNSc:       "http://schemas.openxmlformats.org/drawingml/2006/chart",
		XMLNSlc:      "http://schemas.openxmlformats.org/drawingml/2006/lockedCanvas",
		XMLNSdgm:     "http://schemas.openxmlformats.org/drawingml/2006/diagram",
		XMLNSwps:     "http://schemas.microsoft.com/office/word/2010/wordprocessingShape",
		XMLNSwpg:     "http://schemas.microsoft.com/office/word/2010/wordprocessingGroup",
		XMLNSw14:     "http://schemas.microsoft.com/office/word/2010/wordml",
		XMLNSw15:     "http://schemas.microsoft.com/office/word/2012/wordml",
		XMLNSw16:     "http://schemas.microsoft.com/office/word/2018/wordml",
		XMLNSw16cex:  "http://schemas.microsoft.com/office/word/2018/wordml/cex",
		XMLNSw16cid:  "http://schemas.microsoft.com/office/word/2016/wordml/cid",
		XMLNScr:      "http://schemas.microsoft.com/office/comments/2020/reactions",
		Namespace:    "http://schemas.microsoft.com/office/tasks/2019/documenttasks",
	}
}

func (n *Numbering) ClearNumbering() error {
	n.AbstractNums = make([]*AbstractNum, 0, 10)
	n.Nums = make([]*Num, 0, 10)
	return nil
}

func (n *Numbering) HasNumberingLevel(level int) bool {
	for _, num := range n.Nums {
		if num.NumId == level {
			return true
		}
	}
	return false
}

func (n *Numbering) SetNumberingLevelOrder(level int, isOrdered bool) error {
	if !n.HasNumberingLevel(level) {
		return nil // No change needed if the level does not exist
	}

	for i, abstractNum := range n.AbstractNums {
		if abstractNum.AbstractNumId == level {
			if isOrdered {
				n.AbstractNums[i].Levels = OrderedLevels
			} else {
				n.AbstractNums[i].Levels = UnorderedLevels
			}
			return nil
		}
	}

	return nil // Level not found
}

func (n *Numbering) InsertNewNumberingLevel(isOrdered bool) error {
	newNumId := len(n.AbstractNums) + 1
	n.AbstractNums = append(n.AbstractNums, NewAbstractNumWithOrdering(newNumId, isOrdered))
	n.Nums = append(n.Nums, NewNum(newNumId))
	return nil
}

func (n *Numbering) SetNumberingLevel(level int, isOrdered bool) error {
	if !n.HasNumberingLevel(level) {
		return fmt.Errorf("numbering level %d does not exist", level)
	}
	var levels []Level
	if isOrdered {
		levels = OrderedLevels
	} else {
		levels = UnorderedLevels
	}
	n.AbstractNums[level-1].Levels = levels
	return nil
}

// This element specifies the contents of a main document part in a WordprocessingML document.
type AbstractNum struct {
	XMLName       xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:abstractNum"`
	AbstractNumId int      `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:abstractNumId,attr"`
	Levels        []Level  `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:lvl"`
}

func NewAbstractNum(abstractNumId int) *AbstractNum {
	return &AbstractNum{
		AbstractNumId: abstractNumId,
	}
}

func NewAbstractNumWithOrdering(abstractNumId int, isOrdered bool) *AbstractNum {
	if isOrdered {
		return &AbstractNum{
			AbstractNumId: abstractNumId,
			Levels:        OrderedLevels,
		}
	} else {
		return &AbstractNum{
			AbstractNumId: abstractNumId,
			Levels:        UnorderedLevels,
		}
	}
}

type Num struct {
	XMLName       xml.Name      `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:num"`
	NumId         int           `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:numId,attr"`
	AbstractNumId AbstractNumId `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:abstractNumId"`
}

func NewNum(level int) *Num {
	return &Num{
		NumId:         level,
		AbstractNumId: *NewAbstractNumId(level),
	}
}

type AbstractNumId struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:abstractNumId"`
	Val     int      `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:val,attr"`
}

func NewAbstractNumId(val int) *AbstractNumId {
	return &AbstractNumId{
		Val: val,
	}
}

type Level struct {
	XMLName   xml.Name  `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:lvl"`
	Level     int       `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:ilvl,attr"`
	Start     Start     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:start"`
	NumFmt    NumFmt    `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:numFmt"`
	LevelText LevelText `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:lvlText"`
}

type Start struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:start"`
	Val     int      `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:val,attr"`
}

type LevelText struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:lvlText"`
	Val     string   `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:val,attr"`
}

type LevelJustification struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:lvlJc"`
	Val     string   `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:val,attr"`
}

type NumFmt struct {
	XMLName xml.Name     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:numFmt"`
	Val     types.NumFmt `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main w:val,attr"`
}

var OrderedLevelText = LevelText{Val: "%1."}
var OrderedLevels = []Level{
	{
		Level:     0,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     1,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     2,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     3,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     4,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     5,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     6,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     7,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
	{
		Level:     8,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtDecimal},
		LevelText: OrderedLevelText,
	},
}

var UnorderedLevelText = LevelText{Val: "●"}
var UnorderedLevels = []Level{
	{
		Level:     0,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     1,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     2,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     3,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     4,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     5,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     6,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     7,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
	{
		Level:     8,
		Start:     Start{Val: 1},
		NumFmt:    NumFmt{Val: types.NumFmtBullet},
		LevelText: UnorderedLevelText,
	},
}
