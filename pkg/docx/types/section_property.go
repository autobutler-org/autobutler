package types

import (
	"github.com/nbio/xml"
)

// Document Final Section Properties : w:sectPr
type SectionProperty struct {
	XMLName xml.Name `xml:"w:sectPr"`

	HeaderReference *HeaderReference                `xml:"headerReference,omitempty"`
	FooterReference *FooterReference                `xml:"footerReference,omitempty"`
	PageSize        *PageSize                       `xml:"pgSz,omitempty"`
	Type            *GenSingleStrVal[SectionMark]   `xml:"type,omitempty"`
	PageMargin      *PageMargin                     `xml:"pgMar,omitempty"`
	PageNum         *PageNumberingType              `xml:"pgNumType,omitempty"`
	FormProt        *GenSingleStrVal[OnOffValue]    `xml:"formProt,omitempty"`
	TitlePg         *GenSingleStrVal[OnOffValue]    `xml:"titlePg,omitempty"`
	TextDir         *GenSingleStrVal[TextDirection] `xml:"textDirection,omitempty"`
	DocGrid         *DocGrid                        `xml:"docGrid,omitempty"`
}

func NewSectionProperty() *SectionProperty {
	return &SectionProperty{
		PageSize:   NewPageSize(),
		PageMargin: NewPageMargin(),
		PageNum:    NewPageNumberingType(),
	}
}
