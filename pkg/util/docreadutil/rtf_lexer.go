package docreadutil

// cspell:ignore nback

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// The bounds the RTF specification sets on a control word: at most 32
// letters, and a parameter of at most 10 digits that fits a signed 32-bit
// integer. Longer ones are read through and cut to these.
const (
	maxRTFWordLength  = 32
	maxRTFParamDigits = 10
)

// rtfTokenKind is what an [rtfToken] is.
type rtfTokenKind uint8

const (
	// rtfEOF is the end of the file.
	rtfEOF rtfTokenKind = iota
	// rtfOpen and rtfClose are a group's braces.
	rtfOpen
	rtfClose
	// rtfControl is a control word, or a control symbol other than an
	// escaped character: \* and \~ are control words named "*" and "~".
	rtfControl
	// rtfByte is one byte of text: a literal byte, an escaped \\, \{ or
	// \}, or a \'hh hex escape.
	rtfByte
)

// rtfToken is one token of an RTF file.
type rtfToken struct {
	kind rtfTokenKind
	// word is a control word's meaning; zero for one the reader does
	// not act on.
	word rtfWord
	// param is a control word's parameter, and hasParam whether it had one.
	param    int
	hasParam bool
	// b is an rtfByte's byte.
	b byte
}

// rtfLexer splits an RTF file into tokens, a byte at a time from a buffered
// reader, so a file of any size costs one buffer.
type rtfLexer struct {
	r *bufio.Reader
	// back holds bytes read ahead and put back, the last put back first.
	back  [2]byte
	nback int
	// name holds the control word being read.
	name [maxRTFWordLength]byte
}

// newRTFLexer returns a lexer reading r.
func newRTFLexer(r io.Reader) *rtfLexer {
	return &rtfLexer{r: bufio.NewReaderSize(r, 32<<10)}
}

// readByte reads the next byte, returning io.EOF at the end.
func (l *rtfLexer) readByte() (byte, error) {
	if l.nback > 0 {
		l.nback--
		return l.back[l.nback], nil
	}
	return l.r.ReadByte()
}

// unread puts a byte back; at most two are ever put back at once.
func (l *rtfLexer) unread(b byte) {
	l.back[l.nback] = b
	l.nback++
}

// discard drops the next n bytes without holding them.
func (l *rtfLexer) discard(n int64) error {
	for ; n > 0 && l.nback > 0; n-- {
		l.nback--
	}
	_, err := io.CopyN(io.Discard, l.r, n)
	return err
}

// next reads the next token. Line breaks outside control words are not text
// in RTF, and are dropped here. A read error other than the end of the file
// is returned as it is.
func (l *rtfLexer) next() (rtfToken, error) {
	for {
		b, err := l.readByte()
		if err != nil {
			return eofToken(err)
		}
		switch b {
		case '{':
			return rtfToken{kind: rtfOpen}, nil
		case '}':
			return rtfToken{kind: rtfClose}, nil
		case '\r', '\n':
			continue
		case '\\':
			return l.control()
		}
		return rtfToken{kind: rtfByte, b: b}, nil
	}
}

// eofToken turns the end of the file into the rtfEOF token, and passes any
// other error on.
func eofToken(err error) (rtfToken, error) {
	if errors.Is(err, io.EOF) {
		return rtfToken{kind: rtfEOF}, nil
	}
	return rtfToken{}, err
}

// control reads what follows a backslash.
func (l *rtfLexer) control() (rtfToken, error) {
	b, err := l.readByte()
	if err != nil {
		return eofToken(err)
	}
	switch {
	case isASCIILetter(b):
		return l.word(b)
	case b == '\\' || b == '{' || b == '}':
		return rtfToken{kind: rtfByte, b: b}, nil
	case b == '\'':
		return l.hex()
	case b == '\r' || b == '\n':
		// A backslash before a line break is a paragraph mark.
		return rtfToken{kind: rtfControl, word: wPar}, nil
	}
	return rtfToken{kind: rtfControl, word: rtfSymbols[b]}, nil
}

// word reads a control word whose first letter is first, its parameter,
// and the space that may end it.
func (l *rtfLexer) word(first byte) (rtfToken, error) {
	l.name[0] = first
	n := 1
	var b byte
	var err error
	for {
		if b, err = l.readByte(); err != nil || !isASCIILetter(b) {
			break
		}
		if n < len(l.name) {
			l.name[n] = b
			n++
		}
	}
	tok := rtfToken{kind: rtfControl, word: rtfWords[string(l.name[:n])]}
	if err != nil {
		return tok, ignoreEOF(err)
	}

	negative := false
	if b == '-' {
		next, err := l.readByte()
		if err != nil {
			l.unread(b)
			return tok, ignoreEOF(err)
		}
		if !isASCIIDigit(next) {
			// A hyphen that is not a sign is text after the word.
			l.unread(next)
			l.unread(b)
			return tok, nil
		}
		negative, b = true, next
	}
	if isASCIIDigit(b) {
		tok.hasParam = true
		digits := 0
		for ; err == nil && isASCIIDigit(b); b, err = l.readByte() {
			if digits < maxRTFParamDigits {
				tok.param = tok.param*10 + int(b-'0')
				digits++
			}
		}
		tok.param = min(tok.param, 1<<31-1)
		if negative {
			tok.param = -tok.param
		}
		if err != nil {
			return tok, ignoreEOF(err)
		}
	}
	if b != ' ' {
		l.unread(b)
	}
	if tok.word == wBin && tok.hasParam && tok.param > 0 {
		// \binN is followed by N bytes of binary data, which may hold
		// anything, braces included. It is discarded as it is read, and
		// never held.
		if err := l.discard(int64(tok.param)); err != nil {
			return tok, ignoreEOF(err)
		}
	}
	return tok, nil
}

// skipPlain discards the bytes up to the next backslash or brace, a
// buffer at a time: inside a group being skipped, that is all of them.
func (l *rtfLexer) skipPlain() error {
	if l.nback > 0 {
		return nil
	}
	for {
		if l.r.Buffered() == 0 {
			if _, err := l.r.Peek(1); err != nil {
				return ignoreEOF(err)
			}
		}
		buf, _ := l.r.Peek(l.r.Buffered())
		if i := bytes.IndexAny(buf, `\{}`); i >= 0 {
			_, err := l.r.Discard(i)
			return err
		}
		if _, err := l.r.Discard(len(buf)); err != nil {
			return err
		}
	}
}

// hex reads the two hex digits of a \'hh escape. An escape without two
// digits is dropped.
func (l *rtfLexer) hex() (rtfToken, error) {
	var v byte
	for range 2 {
		b, err := l.readByte()
		if err != nil {
			return eofToken(err)
		}
		d, ok := hexDigit(b)
		if !ok {
			// The escape is dropped, and what broke it is read again as
			// what it is.
			l.unread(b)
			return rtfToken{kind: rtfControl}, nil
		}
		v = v<<4 | d
	}
	return rtfToken{kind: rtfByte, b: v}, nil
}

// ignoreEOF drops io.EOF, which the next read meets again.
func ignoreEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func isASCIILetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

func isASCIIDigit(b byte) bool { return b >= '0' && b <= '9' }

// hexDigit is the value of a hex digit.
func hexDigit(b byte) (byte, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}
