package docreadutil

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// children reads the children of the element whose start was just read,
// through to its end. fn gets each child element's start and must read the
// child through to its end, by reading its children or skipping it.
func children(src tokenSource, fn func(xml.StartElement) error) error {
	for {
		tok, err := src.Token()
		if err != nil {
			return xmlError(err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if err := fn(t); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

// text reads the character data of the element whose start was just read,
// through to its end, skipping any child elements.
func text(src tokenSource) (string, error) {
	var sb strings.Builder
	for {
		tok, err := src.Token()
		if err != nil {
			return "", xmlError(err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			sb.Write(t)
		case xml.StartElement:
			if err := src.Skip(); err != nil {
				return "", xmlError(err)
			}
		case xml.EndElement:
			return sb.String(), nil
		}
	}
}

// root reads up to a part's root element and checks its local name.
func root(src tokenSource, part string, want ...string) (xml.StartElement, error) {
	for {
		tok, err := src.Token()
		if err != nil {
			return xml.StartElement{}, xmlError(err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			for _, w := range want {
				if start.Name.Local == w {
					return start, nil
				}
			}
			return xml.StartElement{}, fmt.Errorf("%w: %s is not a %s part", ErrMalformed, part, want[0])
		}
	}
}

// xmlError makes a decoding error one Convert reports: limits and malformed
// parts as they are, and a part that ends before its elements do as
// malformed.
func xmlError(err error) error {
	switch {
	case errors.Is(err, ErrTooLarge), errors.Is(err, ErrMalformed), errors.Is(err, errTruncated):
		return err
	case errors.Is(err, io.EOF):
		return fmt.Errorf("%w: a part ends before its elements do", ErrMalformed)
	}
	return fmt.Errorf("%w: %v", ErrMalformed, err)
}

// attr returns the value of the element's attribute with the local name,
// whatever its namespace.
func attr(start xml.StartElement, local string) string {
	for _, a := range start.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// heldXML opens a part to be read and held whole, as a decoder that fails
// past maxBytes or past officezip's default depth. Close the returned
// closer when done.
func heldXML(pkg *officezip.Package, name string, maxBytes int64) (*xml.Decoder, io.Closer, error) {
	rc, err := pkg.Part(name)
	if err != nil {
		return nil, nil, err
	}
	counter := &depthCounter{
		dec:  xml.NewDecoder(&capReader{r: rc, name: name, n: maxBytes}),
		name: name,
		max:  officezip.DefaultMaxDepth,
	}
	return xml.NewTokenDecoder(counter), rc, nil
}

func (c *depthCounter) Token() (xml.Token, error) {
	if c.err != nil {
		return nil, c.err
	}
	tok, err := c.dec.Token()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}
		return nil, xmlError(err)
	}
	switch tok.(type) {
	case xml.StartElement:
		c.depth++
		if c.depth > c.max {
			c.err = fmt.Errorf("%w: %s nests XML more than %d elements deep", ErrTooLarge, c.name, c.max)
			return nil, c.err
		}
	case xml.EndElement:
		c.depth--
	}
	return tok, nil
}

func (c *capReader) Read(buf []byte) (int, error) {
	if c.read >= c.n {
		// One byte more is what tells a part that is too large from one
		// that fills the cap exactly.
		var probe [1]byte
		if n, err := c.r.Read(probe[:]); n == 0 {
			return 0, err
		}
		return 0, fmt.Errorf("%w: %s is larger than %d bytes", ErrTooLarge, c.name, c.n)
	}
	if int64(len(buf)) > c.n-c.read {
		buf = buf[:c.n-c.read]
	}
	n, err := c.r.Read(buf)
	c.read += int64(n)
	return n, err
}
