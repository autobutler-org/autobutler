package quill

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"autobutler/pkg/docx"
	"autobutler/pkg/docx/types"
)

func (d Delta) SaveDocxFile(filename string) error {
	doc, err := d.ToDocx()
	if err != nil {
		return fmt.Errorf("failed to convert delta to docx: %w", err)
	}
	writer, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for saving: %w", err)
	}
	defer writer.Close()
	_, err = doc.WriteTo(writer)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func FromDocx(filename string) (Delta, error) {
	delta := Delta{}
	doc, err := docx.OpenDocument(filename)
	if err != nil {
		return delta, fmt.Errorf("failed to open document: %w", err)
	}
	relationships := doc.Document.DocRels.Relationships
	if relationships == nil {
		relationships = []*docx.Relationship{}
	}
	for i, child := range doc.Document.Body.Children {
		switch para := child.(type) {
		case *docx.Paragraph:
			style, _ := para.GetStyle()
			var stylingOp *Op = nil
			if style != nil {
				if headerLevelStr, found := strings.CutPrefix(style.Name.Val, "heading "); found {
					headerLevel, err := strconv.Atoi(headerLevelStr)
					if err != nil {
						return delta, fmt.Errorf("failed to parse header level from style name: %w", err)
					}
					stylingOp = &Op{
						Insert: "\n",
						Attributes: map[string]interface{}{
							"header": headerLevel,
						},
					}
				}
			}
			property := para.Property
			//#region AI SLOP
			// TODO: come back and actually understand this
			// Only add a list stylingOp if the paragraph is actually part of a valid list
			if property != nil && property.NumProp != nil && property.NumProp.NumID != nil &&
				property.NumProp.NumID.Val > 0 &&
				property.NumProp.ILvl != nil {

				numID := property.NumProp.NumID.Val
				iLvl := property.NumProp.ILvl.Val
				isOrdered := false
				hasList := false

				if numID-1 < len(doc.Numbering.AbstractNums) && numID-1 >= 0 {
					for _, level := range doc.Numbering.AbstractNums[numID-1].Levels {
						if level.Level == iLvl {
							hasList = true
							isOrdered = level.NumFmt.Val == types.NumFmtDecimal
							if err := doc.Numbering.SetNumberingLevel(numID, isOrdered); err != nil {
								return delta, fmt.Errorf("failed to set numbering level: %w", err)
							}
							break
						}
					}
				}

				if hasList {
					var attrName string
					if isOrdered {
						attrName = "ordered"
					} else {
						attrName = "bullet"
					}
					stylingOp = &Op{
						Insert: "\n",
						Attributes: map[string]any{
							"indent": iLvl,
							"list":   attrName,
						},
					}
				}
			}
			//#endregion AI SLOP
			for j, paraChild := range para.Children {
				var op Op = Op{}
				var run *docx.Run = nil
				switch paraChild := paraChild.(type) {
				case *docx.Hyperlink:
					if paraChild.Id == "" {
						return delta, fmt.Errorf("hyperlink ID is empty in paragraph %d, child %d", i, j)
					}
					relationship := findRelationshipByID(relationships, paraChild.Id)
					op.Attributes = map[string]interface{}{
						"link": relationship.Target,
					}
					run = paraChild.Run
				case *docx.Run:
					run = paraChild
				}
				property := run.Property
				hadRun := len(run.Children) > 0
				for _, runChild := range run.Children {
					switch runChild := runChild.(type) {
					case *types.Text:
						op.Insert = runChild.Text
						if property != nil {
							if op.Attributes == nil {
								op.Attributes = make(map[string]interface{})
							}
							if property.Bold != nil {
								op.Attributes["bold"] = true
							}
							if property.Italic != nil {
								op.Attributes["italic"] = true
							}
							if property.Underline != nil &&
								property.Underline.Val != types.UnderlineNone {
								op.Attributes["underline"] = true
							}
							if property.Color != nil {
								op.Attributes["color"] = property.Color.Val
							}
						}
					default:
						return delta, fmt.Errorf("unsupported run child type")
					}
				}
				delta.Ops = append(delta.Ops, op)
				if hadRun && stylingOp != nil {
					delta.Ops = append(delta.Ops, *stylingOp)
				}
			}
		default:
			return delta, fmt.Errorf("unsupported body child type at index %d: %T", i, child)
		}
	}
	return delta, nil
}

type opParsingState struct {
	abstractNumId int
}

func (d Delta) ToDocx() (*docx.RootDoc, error) {
	doc, err := docx.NewRootDocument()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize new document: %w", err)
	}
	doc.Numbering.ClearNumbering()

	numOps := len(d.Ops)
	parsingState := opParsingState{
		abstractNumId: 1,
	}
	for i := 0; i < numOps; i++ {
		op := d.Ops[i]
		var stylingOp *Op = nil
		if i < numOps-1 && d.Ops[i+1].isDocxStylingOp() {
			stylingOp = &d.Ops[i+1]
		}
		if (stylingOp != nil && stylingOp.isDocxListOp()) && len(op.Insert) > 1 && strings.HasPrefix(op.Insert, "\n") {
			parsingState.abstractNumId++
		}
		err := op.toDocx(doc, stylingOp, &parsingState)
		if err != nil {
			return nil, fmt.Errorf("failed to generate docx from delta op: %w", err)
		}
		if stylingOp != nil {
			i++
		}
	}
	return doc, nil
}

func findRelationshipByID(relationships []*docx.Relationship, id string) *docx.Relationship {
	for _, rel := range relationships {
		if rel.ID == id {
			return rel
		}
	}
	return nil
}

func (o Op) isDocxStylingOp() bool {
	if o.Attributes == nil {
		return false
	}
	stylingAttributes := []string{"list", "header"}
	for _, v := range stylingAttributes {
		if _, ok := o.Attributes[v]; ok {
			return true
		}
	}
	return false
}

func (o Op) isDocxListOp() bool {
	if o.Attributes == nil {
		return false
	}
	if listKind, ok := o.Attributes["list"].(string); ok && listKind != "" {
		return true
	}
	if _, ok := o.Attributes["indent"]; ok {
		return true
	}
	return false
}

type stylable[T any] interface {
	Bold(bool) T
	Italic(bool) T
	Underline(types.Underline) T
	Color(string) T
}

func style[T stylable[T]](obj T, attributes map[string]interface{}) {
	if attributes == nil {
		return
	}
	if bold, ok := attributes["bold"].(bool); ok && bold {
		obj.Bold(bold)
	}
	if italic, ok := attributes["italic"].(bool); ok && italic {
		obj.Italic(italic)
	}
	if underline, ok := attributes["underline"].(bool); ok && underline {
		obj.Underline(types.UnderlineSingle)
	}
	if color, ok := attributes["color"].(string); ok && color != "" {
		obj.Color(strings.ToUpper(strings.TrimPrefix(color, "#")))
	}
}

func (o Op) toDocx(doc *docx.RootDoc, stylingOp *Op, parsingState *opParsingState) error {
	var p *docx.Paragraph = nil
	if stylingOp != nil {
		if headerLevel, ok := stylingOp.Attributes["header"].(float64); ok && headerLevel > 0 && headerLevel < 10 {
			if link, ok := o.Attributes["link"].(string); ok {
				_, hyperLink, err := doc.AddHeadingHyperlink(o.Insert, link, uint(headerLevel))
				if err != nil {
					return err
				}
				if bold, ok := o.Attributes["bold"].(bool); ok && bold {
					hyperLink.Bold(true)
				}
				if italic, ok := o.Attributes["italic"].(bool); ok && italic {
					hyperLink.Italic(italic)
				}
				if underline, ok := o.Attributes["underline"].(bool); ok && underline {
					hyperLink.Underline(types.UnderlineSingle)
				}
				if color, ok := o.Attributes["color"].(string); ok && color != "" {
					hyperLink.Color(strings.ToUpper(strings.TrimPrefix(color, "#")))
				}
				style(hyperLink, o.Attributes)
			} else {
				_, run, err := doc.AddHeading(o.Insert, uint(headerLevel))
				if err != nil {
					return err
				}
				style(run, o.Attributes)
			}
			return nil
		}
		if listKind, ok := stylingOp.Attributes["list"].(string); ok && listKind != "" {
			indentF := 0.0
			if indentF, ok = stylingOp.Attributes["indent"].(float64); !ok {
				indentF = 0
			}
			indent := int(indentF)
			if indent < 0 || indent > 8 {
				return fmt.Errorf("invalid list indent level: %d, must be between 0 and 8, inclusive", indent)
			}
			switch listKind {
			case "ordered":
				currentLevel := len(doc.Numbering.AbstractNums)
				for i := currentLevel; i < parsingState.abstractNumId; i++ {
					doc.Numbering.InsertNewNumberingLevel(true)
				}
			case "bullet":
				currentLevel := len(doc.Numbering.AbstractNums)
				for i := currentLevel; i < parsingState.abstractNumId; i++ {
					doc.Numbering.InsertNewNumberingLevel(false)
				}
			default:
				return fmt.Errorf("unknown list kind: %s", listKind)
			}
			p = doc.AddEmptyParagraph()
			pPr := types.NewParagraphProperty()
			pPr.NumProp = &types.NumberingProperty{
				NumID: types.NewDecimalNum(parsingState.abstractNumId),
				ILvl:  types.NewDecimalNum(indent),
			}
			p.Property = pPr

		}
	}
	if o.Attributes == nil {
		o.Attributes = make(map[string]interface{})
	}
	if p == nil {
		p = doc.AddEmptyParagraph()
	}
	if link, ok := o.Attributes["link"].(string); ok {
		hyperLink := p.AddLink(o.Insert, link)
		if bold, ok := o.Attributes["bold"].(bool); ok && bold {
			hyperLink.Bold(true)
		}
		if italic, ok := o.Attributes["italic"].(bool); ok && italic {
			hyperLink.Italic(italic)
		}
		if underline, ok := o.Attributes["underline"].(bool); ok && underline {
			hyperLink.Underline(types.UnderlineSingle)
		}
		if color, ok := o.Attributes["color"].(string); ok && color != "" {
			hyperLink.Color(strings.ToUpper(strings.TrimPrefix(color, "#")))
		}
	} else {
		run := p.AddText(o.Insert)
		style(run, o.Attributes)
	}
	return nil
}
