package odputil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

// pictureFormats maps a sniffed picture type to the extension it is stored
// under: the formats the editor shows.
var pictureFormats = map[string]string{
	"image/png": "png", "image/jpeg": "jpg", "image/gif": "gif", "image/bmp": "bmp", "image/webp": "webp",
}

// pictures stores the first of a frame's pictures the editor can show —
// LibreOffice saves an SVG with a PNG beside it to fall back on — and
// returns its reference. When none can be stored, why the first could not
// is the warning; only a failure to store one at all is an error.
func (s *slideReader) pictures(hrefs []string) (string, bool, error) {
	why := ""
	for _, href := range hrefs {
		ref, reason, err := s.picture(href)
		if err != nil {
			return "", false, err
		}
		if ref != "" {
			return ref, true, nil
		}
		if why == "" {
			why = reason
		}
	}
	if why != "" {
		s.warn(why)
	}
	return "", false, nil
}

// picture stores the picture at href, once per package, and returns its
// reference or why it was left out.
func (s *slideReader) picture(href string) (string, string, error) {
	part, ok := internalPart(href)
	if !ok {
		return "", "Linked pictures are not imported; only pictures saved in the file are.", nil
	}
	if !s.pkg.Has(part) {
		return "", "Some pictures were missing from the file.", nil
	}
	key := strings.ToLower(part)
	if ref, seen := s.media[key]; seen {
		return ref, s.skipped[key], nil
	}
	ref, why, err := s.storePicture(part)
	if err != nil {
		return "", "", err
	}
	s.media[key] = ref
	if ref == "" {
		s.skipped[key] = why
		return "", why, nil
	}
	s.result.Pictures++
	return ref, "", nil
}

// internalPart is the package entry an xlink:href names, or not ok for a
// link out of the package: a URL, an absolute path, or one that climbs out.
func internalPart(href string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" || strings.Contains(href, ":") || strings.HasPrefix(href, "/") || strings.Contains(href, "\\") {
		return "", false
	}
	clean := path.Clean(strings.TrimPrefix(href, "./"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

// storePicture streams one picture to StoreMedia. It returns the stored
// reference, or why the picture was left out.
func (s *slideReader) storePicture(part string) (string, string, error) {
	if s.params.StoreMedia == nil {
		return "", "Pictures are not imported here.", nil
	}
	tooLarge := fmt.Sprintf("Pictures larger than %d MiB were left out.", MaxPartBytes>>20)
	if s.mediaBytes >= MaxMediaBytes {
		return "", fmt.Sprintf("Pictures past the first %d MiB were left out.", MaxMediaBytes>>20), nil
	}
	rc, err := s.pkg.Part(part)
	if errors.Is(err, ErrTooLarge) {
		return "", tooLarge, nil
	}
	if err != nil {
		return "", "Some pictures could not be read.", nil
	}
	defer rc.Close()
	counted := &countingReader{r: rc}

	head := make([]byte, 512)
	n, err := io.ReadFull(counted, head)
	switch {
	case errors.Is(err, ErrTooLarge):
		return "", tooLarge, nil
	case err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF):
		return "", "Some pictures could not be read.", nil
	}
	head = head[:n]
	ext, ok := pictureFormats[http.DetectContentType(head)]
	if !ok {
		format := strings.ToUpper(strings.TrimPrefix(path.Ext(part), "."))
		if format == "" {
			format = "unknown"
		}
		return "", fmt.Sprintf("Pictures in a format the editor cannot show (%s) were left out.", format), nil
	}
	ref, err := s.params.StoreMedia(part, mediaName(part, ext), io.MultiReader(bytes.NewReader(head), counted))
	s.mediaBytes += counted.read
	switch {
	case errors.Is(err, ErrTooLarge):
		return "", tooLarge, nil
	case errors.Is(err, ErrMalformed):
		return "", "Some pictures could not be read.", nil
	case err != nil:
		return "", "", err
	}
	return ref, "", nil
}

// countingReader counts what is read through it.
type countingReader struct {
	r    io.Reader
	read int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += int64(n)
	return n, err
}
