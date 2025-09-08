package types

func NewText(text string) *Text {
	return &Text{
		Text:  text,
		Space: TextSpacePreserve,
	}
}

type TextSpace string

const (
	TextSpaceDefault  TextSpace = "default"
	TextSpacePreserve TextSpace = "preserve"
)
