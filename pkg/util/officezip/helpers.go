package officezip

import (
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

// withDefaults fills every zero limit with its default.
func (l Limits) withDefaults() Limits {
	orDefault := func(v, def int64) int64 {
		if v <= 0 {
			return def
		}
		return v
	}
	l.MaxEntries = int(orDefault(int64(l.MaxEntries), DefaultMaxEntries))
	l.MaxDirectoryBytes = orDefault(l.MaxDirectoryBytes, DefaultMaxDirectoryBytes)
	l.MaxPartBytes = orDefault(l.MaxPartBytes, DefaultMaxPartBytes)
	l.MaxTotalBytes = orDefault(l.MaxTotalBytes, DefaultMaxTotalBytes)
	l.MaxElements = int(orDefault(int64(l.MaxElements), DefaultMaxElements))
	l.MaxDepth = int(orDefault(int64(l.MaxDepth), DefaultMaxDepth))
	return l
}

// zipDirectory reads the entry count and central directory size from the end
// of the archive, following the zip64 locator when the plain record defers to
// it.
func zipDirectory(src io.ReaderAt, size int64) (entries, dirSize uint64, err error) {
	tailLen := min(size, eocdLength+maxZipCommentLength)
	tail := make([]byte, tailLen)
	if _, err := src.ReadAt(tail, size-tailLen); err != nil && !errors.Is(err, io.EOF) {
		return 0, 0, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	at := -1
	for i := len(tail) - eocdLength; i >= 0; i-- {
		if binary.LittleEndian.Uint32(tail[i:]) == eocdSignature {
			at = i
			break
		}
	}
	if at < 0 {
		return 0, 0, fmt.Errorf("%w: not a zip archive", ErrMalformed)
	}
	eocd := tail[at:]
	entries = uint64(binary.LittleEndian.Uint16(eocd[10:]))
	dirSize = uint64(binary.LittleEndian.Uint32(eocd[12:]))
	if entries != 0xFFFF && dirSize != 0xFFFFFFFF {
		return entries, dirSize, nil
	}

	eocdOffset := size - tailLen + int64(at)
	if eocdOffset < zip64LocatorLength {
		return 0, 0, fmt.Errorf("%w: a zip64 archive without its locator", ErrMalformed)
	}
	locator := make([]byte, zip64LocatorLength)
	if _, err := src.ReadAt(locator, eocdOffset-zip64LocatorLength); err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if binary.LittleEndian.Uint32(locator) != zip64LocatorSig {
		return 0, 0, fmt.Errorf("%w: a zip64 archive without its locator", ErrMalformed)
	}
	recordOffset := binary.LittleEndian.Uint64(locator[8:])
	if size < zip64EOCDLength || recordOffset > uint64(size-zip64EOCDLength) {
		return 0, 0, fmt.Errorf("%w: the zip64 record lies outside the file", ErrMalformed)
	}
	record := make([]byte, zip64EOCDLength)
	if _, err := src.ReadAt(record, int64(recordOffset)); err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if binary.LittleEndian.Uint32(record) != zip64EOCDSignature {
		return 0, 0, fmt.Errorf("%w: the zip64 record is missing", ErrMalformed)
	}
	return binary.LittleEndian.Uint64(record[32:]), binary.LittleEndian.Uint64(record[40:]), nil
}

// safePartName reports whether a zip entry name stays inside the package: a
// relative, forward-slash path with no "." or ".." segment and nothing a
// filesystem would read as a drive or a device.
func safePartName(name string) bool {
	if name == "" || len(name) > 1024 || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
		return false
	}
	for _, seg := range strings.Split(strings.TrimSuffix(name, "/"), "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

// resolveTarget resolves a relationship target against the part that holds
// it; an absolute target is from the package root. A target that climbs out
// of the package resolves to nothing.
func resolveTarget(source, target string) (string, bool) {
	if unescaped, err := url.PathUnescape(target); err == nil {
		target = unescaped
	}
	if target == "" || strings.ContainsAny(target, "\\\x00") {
		return "", false
	}
	var kept []string
	if dir := path.Dir(source); !strings.HasPrefix(target, "/") && dir != "." {
		kept = strings.Split(dir, "/")
	}
	for _, seg := range strings.Split(target, "/") {
		switch seg {
		case "", ".":
		case "..":
			if len(kept) == 0 {
				return "", false
			}
			kept = kept[:len(kept)-1]
		default:
			kept = append(kept, seg)
		}
	}
	if len(kept) == 0 {
		return "", false
	}
	return strings.Join(kept, "/"), true
}

// relsPath is the relationships part of a part; the empty name is the
// package's own.
func relsPath(part string) string {
	if part == "" {
		return "_rels/.rels"
	}
	return path.Join(path.Dir(part), "_rels", path.Base(part)+".rels")
}

// decodeXML decodes a whole XML part into v through the counting decoder.
func (p *Package) decodeXML(name string, v any) error {
	dec, err := p.XML(name)
	if err != nil {
		return err
	}
	defer dec.Close()
	if err := dec.Decode(v); err != nil {
		return partError(name, err)
	}
	return nil
}

// partError reports what went wrong reading a part: a part that broke a
// limit, or a malformed one, says so; anything else is malformed too.
func partError(name string, err error) error {
	if errors.Is(err, ErrTooLarge) || errors.Is(err, ErrMalformed) {
		return err
	}
	return fmt.Errorf("%w: %s: %v", ErrMalformed, name, err)
}

func (b *budgetReader) Read(buf []byte) (int, error) {
	allow := min(b.limit-b.read, b.pkg.remaining)
	if allow <= 0 {
		// One byte more is what tells a part that is too large from one
		// that fills its cap exactly.
		var probe [1]byte
		if n, _ := b.rc.Read(probe[:]); n == 0 {
			return 0, io.EOF
		}
		if b.pkg.remaining <= 0 {
			return 0, fmt.Errorf("%w: the package unpacks to more than %d bytes", ErrTooLarge, b.pkg.limits.MaxTotalBytes)
		}
		return 0, fmt.Errorf("%w: %s is larger than %d bytes", ErrTooLarge, b.name, b.limit)
	}
	if int64(len(buf)) > allow {
		buf = buf[:allow]
	}
	n, err := b.rc.Read(buf)
	b.read += int64(n)
	b.pkg.remaining -= int64(n)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("%w: %s: %v", ErrMalformed, b.name, err)
	}
	return n, err
}

func (b *budgetReader) Close() error { return b.rc.Close() }

func (c *tokenCounter) Token() (xml.Token, error) {
	if c.err != nil {
		return nil, c.err
	}
	tok, err := c.dec.Token()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}
		return nil, partError(c.name, err)
	}
	switch tok.(type) {
	case xml.StartElement:
		c.depth++
		c.elements++
		if c.depth > c.maxDepth {
			c.err = fmt.Errorf("%w: %s nests XML more than %d elements deep", ErrTooLarge, c.name, c.maxDepth)
		} else if c.elements > c.maxElements {
			c.err = fmt.Errorf("%w: %s holds more than %d XML elements", ErrTooLarge, c.name, c.maxElements)
		}
		if c.err != nil {
			return nil, c.err
		}
	case xml.EndElement:
		c.depth--
	}
	return tok, nil
}
