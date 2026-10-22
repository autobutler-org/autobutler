package docpropsutil

// cspell:ignore nofchars

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// maxRTFGroupDepth bounds how deeply the scan follows nested groups.
const maxRTFGroupDepth = 256

// rtfScan reads an RTF's header for its \info group and \generator. It reads
// at most [MaxRTFHeaderBytes] and stops at the end of \info, so the body is
// never scanned.
type rtfScan struct {
	r     *bufio.Reader
	props Properties
}

// readRTF extracts properties from an RTF stream.
func readRTF(r io.Reader) (Properties, error) {
	s := &rtfScan{r: bufio.NewReader(io.LimitReader(r, MaxRTFHeaderBytes))}
	head, err := s.r.Peek(5)
	if err != nil || string(head) != `{\rtf` {
		return Properties{}, fmt.Errorf("%w: not an RTF document", ErrMalformed)
	}
	if err := s.scan(); err != nil {
		return Properties{}, err
	}
	return s.props, nil
}

// scan walks the groups of the header. Destinations it does not want are
// skipped by depth counting alone.
func (s *rtfScan) scan() error {
	depth := 0
	// infoDepth is the depth of the \info group while inside it, else 0.
	infoDepth := 0
	var field *rtfField
	for {
		b, err := s.r.ReadByte()
		if err != nil {
			return nil // the cap or the end of the file; either is fine
		}
		switch b {
		case '{':
			depth++
			if depth > maxRTFGroupDepth {
				return fmt.Errorf("%w: RTF groups nest too deeply", ErrMalformed)
			}
		case '}':
			if field != nil && depth == field.depth {
				s.store(field)
				field = nil
			}
			if infoDepth != 0 && depth == infoDepth {
				return nil
			}
			depth--
		case '\\':
			word, arg, hasArg, err := s.control()
			if err != nil {
				return nil
			}
			switch {
			case word == "info" && infoDepth == 0:
				infoDepth = depth
			case word == "generator" && field == nil:
				field = &rtfField{name: word, depth: depth}
			case infoDepth != 0 && depth == infoDepth+1 && field == nil:
				field = newRTFField(word, depth)
			case infoDepth != 0 && depth == infoDepth && hasArg:
				s.storeCount(word, arg)
			case field != nil && field.date != nil && hasArg:
				field.date[word] = arg
			case field != nil && strings.HasPrefix(word, "\\"):
				field.text.WriteString(word[1:])
			case field != nil && word == "'":
				field.text.WriteRune(rune(arg))
			case field != nil && word == "u" && hasArg:
				field.text.WriteRune(rune(uint16(arg)))
				s.skipFallback()
			}
		default:
			if field != nil && field.date == nil && b != '\r' && b != '\n' {
				s.writeText(field, b)
			}
		}
	}
}

// rtfField is one \info destination being collected: text for \title and its
// kin, or a date's \yr \mo \dy \hr \min parts.
type rtfField struct {
	name  string
	depth int
	text  strings.Builder
	date  map[string]int
}

func newRTFField(word string, depth int) *rtfField {
	f := &rtfField{name: word, depth: depth}
	if word == "creatim" || word == "revtim" {
		f.date = map[string]int{}
	}
	return f
}

// control reads a control word or symbol after a backslash, with its numeric
// argument. A symbol such as \{ comes back as the word "\\{" so the caller
// can treat it as text; \'hh as the word "'" with the byte as its argument.
func (s *rtfScan) control() (word string, arg int, hasArg bool, err error) {
	b, err := s.r.ReadByte()
	if err != nil {
		return "", 0, false, err
	}
	switch {
	case b == '\'':
		hi, _ := s.r.ReadByte()
		lo, _ := s.r.ReadByte()
		v, parseErr := strconv.ParseUint(string([]byte{hi, lo}), 16, 8)
		if parseErr != nil {
			return "'", int('?'), true, nil
		}
		return "'", int(v), true, nil
	case b == '{' || b == '}' || b == '\\':
		return "\\" + string(b), 0, false, nil
	case (b < 'a' || b > 'z') && (b < 'A' || b > 'Z'):
		return string(b), 0, false, nil
	}
	var name strings.Builder
	name.WriteByte(b)
	for {
		c, err := s.r.ReadByte()
		if err != nil {
			return name.String(), 0, false, nil
		}
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			name.WriteByte(c)
			continue
		}
		return s.argument(name.String(), c)
	}
}

// argument reads the optional signed number and the one space delimiter that
// follow a control word, c being the first byte after the letters.
func (s *rtfScan) argument(word string, c byte) (string, int, bool, error) {
	var num strings.Builder
	if c == '-' || c >= '0' && c <= '9' {
		num.WriteByte(c)
		for {
			d, err := s.r.ReadByte()
			if err != nil {
				break
			}
			if d >= '0' && d <= '9' && num.Len() < 12 {
				num.WriteByte(d)
				continue
			}
			if d != ' ' {
				_ = s.r.UnreadByte()
			}
			break
		}
		n, err := strconv.Atoi(num.String())
		if err != nil {
			return word, 0, false, nil
		}
		return word, n, true, nil
	}
	if c != ' ' {
		_ = s.r.UnreadByte()
	}
	return word, 0, false, nil
}

// skipFallback drops the ANSI fallback character that follows \uN.
func (s *rtfScan) skipFallback() {
	b, err := s.r.ReadByte()
	if err != nil {
		return
	}
	if b == '\\' { // \'hh fallback
		if nb, err := s.r.ReadByte(); err == nil && nb == '\'' {
			_, _ = s.r.ReadByte()
			_, _ = s.r.ReadByte()
		} else if err == nil {
			_ = s.r.UnreadByte()
			_ = s.r.UnreadByte()
		}
		return
	}
	if b == '{' || b == '}' {
		_ = s.r.UnreadByte()
	}
}

// writeText appends a literal byte of a field's text; bytes outside ASCII
// are taken as the Latin-1 rune they most often are.
func (s *rtfScan) writeText(f *rtfField, b byte) {
	if f.text.Len() > 4096 {
		return
	}
	if b < utf8.RuneSelf {
		f.text.WriteByte(b)
		return
	}
	f.text.WriteRune(rune(b))
}

// store records a closed \info destination.
func (s *rtfScan) store(f *rtfField) {
	text := strings.TrimSpace(f.text.String())
	switch f.name {
	case "title":
		s.props.Title = text
	case "author":
		s.props.Author = text
	case "subject":
		s.props.Subject = text
	case "generator":
		s.props.Application = strings.TrimSpace(strings.TrimSuffix(text, ";"))
	case "creatim":
		s.props.Created = rtfDate(f.date)
	case "revtim":
		s.props.Modified = rtfDate(f.date)
	}
}

// storeCount records \nofpages, \nofwords and \nofchars.
func (s *rtfScan) storeCount(word string, n int) {
	if n < 0 {
		return
	}
	switch word {
	case "nofpages":
		s.props.Pages = n
	case "nofwords":
		s.props.Words = n
	case "nofchars":
		s.props.Characters = n
	}
}

// rtfDate formats a \creatim or \revtim's parts, empty when it has no year.
func rtfDate(d map[string]int) string {
	if d["yr"] <= 0 {
		return ""
	}
	month, day := max(d["mo"], 1), max(d["dy"], 1)
	t := time.Date(d["yr"], time.Month(month), day, d["hr"], d["min"], d["sec"], 0, time.UTC)
	return t.Format(time.RFC3339)
}
