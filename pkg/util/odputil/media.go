package odputil

// cspell:ignore xlink

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/officezip"
)

// pictureFormats maps a sniffed picture type to the extension it is stored
// under: the formats the editor shows, as pptxutil's import has them.
var pictureFormats = map[string]string{
	"image/png": "png", "image/jpeg": "jpg", "image/gif": "gif", "image/bmp": "bmp", "image/webp": "webp",
}

// sniffBytes is how much of a picture is read to learn its format.
const sniffBytes = 512

// Why a picture is left out.
const (
	inlinePictureWarning     = "Pictures held inside the slide's XML were left out."
	linkedPictureWarning     = "Linked pictures are not imported."
	unreadablePictureWarning = "Pictures that could not be read were left out."
)

// imageCapReader passes a picture through to StoreMedia, failing with
// [ErrTooLarge] past [MaxImageBytes].
type imageCapReader struct {
	r    io.Reader
	read int64
	// past is set once the picture is found to be larger than the cap.
	past bool
}

// imageHref is the package part a draw:image shows, or a marker for one it
// holds inline or links to from outside the package.
func (s *slideReader) imageHref(c xml.StartElement) string {
	href := attr(c, nsXlink, "href")
	if href == "" {
		return inlinePictureWarning
	}
	return href
}

// picture stores the picture at href, a part of the package, and returns
// its reference. One that cannot be stored is reported and comes back not
// ok; only a failure to store it at all is an error.
func (s *slideReader) picture(href string) (string, bool, error) {
	part, warning := packagePart(href)
	if warning == "" && !s.pkg.Has(part) {
		warning = "Pictures missing from the file were left out."
	}
	if warning != "" {
		s.warn(warning)
		return "", false, nil
	}
	key := strings.ToLower(part)
	if ref, seen := s.media[key]; seen {
		if ref == "" {
			s.warn(s.skipped[key])
		}
		return ref, ref != "", nil
	}
	ref, why, err := s.storePicture(part)
	if err != nil {
		return "", false, err
	}
	s.media[key] = ref
	if ref == "" {
		s.skipped[key] = why
		s.warn(why)
		return "", false, nil
	}
	s.result.Pictures++
	return ref, true, nil
}

// packagePart resolves a picture's href to a part of the package, or says
// why it is not one: a picture held inline, or linked from outside, which
// is never fetched.
func packagePart(href string) (string, string) {
	if href == inlinePictureWarning {
		return "", inlinePictureWarning
	}
	if u, err := url.Parse(href); err != nil || u.Scheme != "" || u.Host != "" || strings.HasPrefix(href, "/") {
		return "", linkedPictureWarning
	}
	if unescaped, err := url.PathUnescape(href); err == nil {
		href = unescaped
	}
	clean := path.Clean(strings.TrimPrefix(href, "./"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsAny(clean, "\\\x00") {
		return "", linkedPictureWarning
	}
	return clean, ""
}

// storePicture streams one picture part to StoreMedia. It returns the
// stored reference, or "" and why the picture was left out.
func (s *slideReader) storePicture(part string) (string, string, error) {
	if s.params.StoreMedia == nil {
		return "", "Pictures are not imported here.", nil
	}
	tooLarge := fmt.Sprintf("Pictures larger than %d MiB were left out.", MaxImageBytes>>20)
	if s.mediaBytes >= MaxMediaBytes {
		return "", fmt.Sprintf("Pictures past the first %d MiB were left out.", MaxMediaBytes>>20), nil
	}
	rc, err := s.pkg.Part(part)
	if errors.Is(err, officezip.ErrTooLarge) {
		// The part declares more than any part may hold.
		return "", tooLarge, nil
	}
	if err != nil {
		return "", "", xmlError(err)
	}
	defer rc.Close()
	reader := &imageCapReader{r: rc}
	head := make([]byte, sniffBytes)
	n, err := io.ReadFull(reader, head)
	switch {
	case errors.Is(err, officezip.ErrMalformed):
		return "", unreadablePictureWarning, nil
	case err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF):
		return "", "", xmlError(err)
	}
	head = head[:n]
	ext, ok := pictureFormats[http.DetectContentType(head)]
	if !ok {
		format := strings.ToUpper(strings.TrimPrefix(path.Ext(part), "."))
		if format == "" {
			format = "unknown"
		}
		return "", fmt.Sprintf("Pictures in a format the editor cannot show (%s) were left out.", sanitize(format)), nil
	}
	ref, err := s.params.StoreMedia(mediaName(part, ext), io.MultiReader(bytes.NewReader(head), reader))
	s.mediaBytes += reader.read
	switch {
	case reader.past:
		return "", tooLarge, nil
	case errors.Is(err, officezip.ErrTooLarge):
		return "", "", fmt.Errorf("%w: %v", ErrTooLarge, err)
	case errors.Is(err, officezip.ErrMalformed):
		return "", unreadablePictureWarning, nil
	case err != nil:
		return "", "", err
	}
	return ref, "", nil
}

func (c *imageCapReader) Read(p []byte) (int, error) {
	if c.read >= MaxImageBytes {
		// One byte more is what tells a picture that is too large from one
		// that fills the cap exactly.
		var probe [1]byte
		if n, err := c.r.Read(probe[:]); n == 0 {
			return 0, err
		}
		c.past = true
		return 0, fmt.Errorf("%w: a picture is larger than %d bytes", ErrTooLarge, MaxImageBytes)
	}
	if int64(len(p)) > MaxImageBytes-c.read {
		p = p[:MaxImageBytes-c.read]
	}
	n, err := c.r.Read(p)
	c.read += int64(n)
	return n, err
}

// mediaName is the name a picture part is stored under: its own stem, kept
// to characters every filesystem takes, and the extension of its content.
func mediaName(part, ext string) string {
	stem := strings.TrimSuffix(path.Base(part), path.Ext(part))
	var b strings.Builder
	for _, r := range stem {
		if b.Len() >= 64 {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		b.WriteString("image")
	}
	return b.String() + "." + ext
}
