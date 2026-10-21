package odputil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// header writes everything ahead of the first slide.
func (q *qslideWriter) header(title string, size outSize) error {
	t, err := marshalIndented(title, "  ")
	if err != nil {
		return err
	}
	s, err := marshalIndented(size, "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(q.w, "{\n  \"schemaVersion\": %d,\n  \"title\": %s,\n  \"size\": %s,\n  \"slides\": [",
		importSchemaVersion, t, s)
	return err
}

// slide writes one slide.
func (q *qslideWriter) slide(slide outSlide) error {
	body, err := marshalIndented(slide, "    ")
	if err != nil {
		return err
	}
	sep := ",\n    "
	if q.slides == 0 {
		sep = "\n    "
	}
	q.slides++
	_, err = io.WriteString(q.w, sep+body)
	return err
}

// close ends the slide list and the document.
func (q *qslideWriter) close() error {
	end := "\n  ]\n}\n"
	if q.slides == 0 {
		end = "]\n}\n"
	}
	_, err := io.WriteString(q.w, end)
	return err
}

// marshalIndented is v as indented JSON whose continuation lines carry
// prefix. HTML is not escaped, as dart:convert does not escape it.
func marshalIndented(v any, prefix string) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent(prefix, "  ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}
