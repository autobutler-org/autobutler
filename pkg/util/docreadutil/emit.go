package docreadutil

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// errTruncated stops a reader once an output limit is reached. Convert
// turns it into a truncated result, never an error.
var errTruncated = errors.New("docreadutil: an output limit was reached")

// The nouns warnings and omitted blocks use, singular and plural.
var warningNouns = map[string]string{
	"embedded object":   "embedded objects",
	"embedded document": "embedded documents",
	"chart":             "charts",
	"diagram":           "diagrams",
	"shape":             "shapes",
	"text box":          "text boxes",
	"equation":          "equations",
	"image":             "images",
	"linked image":      "linked images",
}

// Warnings that are not counted, said once.
const (
	warnHeadersFooters = "Headers and footers are not shown"
	warnComments       = "Comments are not shown"
	warnNotes          = "Some notes are not shown"
)

// newEmitter returns an emitter writing what params asks for.
func newEmitter(params ConvertParams) *emitter {
	out := &countingWriter{w: params.Out}
	var w blockWriter
	if params.Output == OutputDelta {
		w = &deltaWriter{w: out}
	} else {
		w = newNDJSONWriter(out)
	}
	return &emitter{
		ctx:    params.Ctx,
		limits: params.Limits.withDefaults(),
		format: params.Format,
		out:    out,
		w:      w,
	}
}

// withDefaults fills every zero limit with its default.
func (l Limits) withDefaults() Limits {
	if l.MaxBlocks <= 0 {
		l.MaxBlocks = DefaultMaxBlocks
	}
	if l.MaxOutputBytes <= 0 {
		l.MaxOutputBytes = DefaultMaxOutputBytes
	}
	if l.MaxHeldBytes <= 0 {
		l.MaxHeldBytes = DefaultMaxHeldBytes
	}
	if l.MaxDefinitions <= 0 {
		l.MaxDefinitions = DefaultMaxDefinitions
	}
	if l.MaxTableDepth <= 0 {
		l.MaxTableDepth = DefaultMaxTableDepth
	}
	return l
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// admit counts one more block against the block limit, before the reader
// reads it. It fails with errTruncated once the limit is reached, and with
// the context's error once the context is done.
func (e *emitter) admit() error {
	if e.ctx != nil {
		if err := e.ctx.Err(); err != nil {
			return err
		}
	}
	if e.blocks >= e.limits.MaxBlocks {
		e.truncated = true
		return errTruncated
	}
	e.blocks++
	return nil
}

// charge counts n more bytes held for output. It fails with errTruncated
// when they would take what is written and held past the output limit.
func (e *emitter) charge(n int) error {
	if e.out.n+e.held+int64(n) > e.limits.MaxOutputBytes {
		e.truncated = true
		return errTruncated
	}
	e.held += int64(n)
	return nil
}

// tally counts a text block's words, characters and paragraphs.
func (e *emitter) tally(b Block) {
	if len(b.Runs) == 0 {
		return
	}
	var sb strings.Builder
	for _, r := range b.Runs {
		sb.WriteString(r.Text)
	}
	text := sb.String()
	e.words += len(strings.Fields(text))
	e.chars += utf8.RuneCountInString(text) - strings.Count(text, "\n")
	if strings.TrimSpace(text) != "" {
		e.paras++
	}
}

// write hands a top-level block to the writer.
func (e *emitter) write(b Block) error {
	if err := e.begin(); err != nil {
		return err
	}
	e.held = 0
	return e.w.block(b)
}

// begin starts the writer, once.
func (e *emitter) begin() error {
	if e.started {
		return nil
	}
	e.started = true
	return e.w.begin(metaLine{Type: "meta", Version: StreamVersion, Format: e.format, Title: e.title})
}

// finish ends the writer.
func (e *emitter) finish() error {
	if err := e.begin(); err != nil {
		return err
	}
	return e.w.end(endLine{
		Type:       "end",
		Truncated:  e.truncated,
		Words:      e.words,
		Characters: e.chars,
		Paragraphs: e.paras,
		Warnings:   e.warnings.list(),
	})
}

// result reports what the conversion read and wrote.
func (e *emitter) result() ConvertResult {
	return ConvertResult{
		Title:      e.title,
		Truncated:  e.truncated,
		Blocks:     e.blocks,
		Words:      e.words,
		Characters: e.chars,
		Paragraphs: e.paras,
		Warnings:   append(e.warnings.list(), e.w.losses()...),
	}
}

// count notes one more of a kind left out.
func (w *warnings) count(kind string) {
	if w.counts == nil {
		w.counts = make(map[string]int)
	}
	if _, seen := w.counts[kind]; !seen {
		w.order = append(w.order, kind)
	}
	w.counts[kind]++
}

// note says a message once.
func (w *warnings) note(message string) {
	if _, seen := w.counts[message]; !seen {
		w.count(message)
	}
}

// list is the warnings as sentences, in the order first seen.
func (w *warnings) list() []string {
	out := make([]string, 0, len(w.order))
	for _, kind := range w.order {
		plural, counted := warningNouns[kind]
		if !counted {
			out = append(out, kind)
			continue
		}
		out = append(out, countPhrase(w.counts[kind], kind, plural)+" not shown")
	}
	return out
}

// countPhrase is "1 table" or "3 tables".
func countPhrase(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}
