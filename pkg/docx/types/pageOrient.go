package types

import (
	"errors"
)

type PageOrient string

const (
	PageOrientPortrait  PageOrient = "portrait"
	PageOrientLandscape PageOrient = "landscape"
)

func PageOrientFromStr(value string) (PageOrient, error) {
	switch value {
	case "portrait":
		return PageOrientPortrait, nil
	case "landscape":
		return PageOrientLandscape, nil
	default:
		return "", errors.New("Invalid Orient Input")
	}
}
