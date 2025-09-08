package types

import (
	"fmt"
)

type Hex string

func NewHex(i uint64) *Hex {
	hexStr := fmt.Sprintf("%08x", i)
	hex := Hex(hexStr)
	return &hex
}

func NewHexFromString(s string) *Hex {
	hex := Hex(s)
	return &hex
}
