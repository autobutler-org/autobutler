package uploadutil

import (
	"fmt"
	"strconv"
	"strings"
)

// contentRangeUnit is the only unit the chunk endpoint accepts. RFC 7233 allows
// others; nothing sends them.
const contentRangeUnit = "bytes "

// ContentRange is a parsed Content-Range request header: `bytes 0-262143/1000000`.
// End is inclusive, matching the header and Google's resumable upload protocol,
// so a chunk carries End-Start+1 bytes.
type ContentRange struct {
	Start int64
	End   int64
	Total int64
}

// Length is the number of bytes the chunk body must carry.
func (r ContentRange) Length() int64 {
	return r.End - r.Start + 1
}

// ParseContentRange reads the header a chunk PUT must carry. Every failure
// wraps ErrInvalidRange, which the HTTP layer turns into a 400: a client that
// cannot state which bytes it is sending has nothing the server can commit.
//
// The `bytes */SIZE` form RFC 7233 allows for asking what the server has is
// deliberately rejected — that question is GET on the session, which answers
// with the offset in the body rather than in a header.
func ParseContentRange(header string) (ContentRange, error) {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return ContentRange{}, fmt.Errorf("%w: missing Content-Range header", ErrInvalidRange)
	}
	if !strings.HasPrefix(trimmed, contentRangeUnit) {
		return ContentRange{}, fmt.Errorf("%w: %q is not a byte range", ErrInvalidRange, header)
	}

	spec := strings.TrimSpace(strings.TrimPrefix(trimmed, contentRangeUnit))
	rangePart, totalPart, ok := strings.Cut(spec, "/")
	if !ok {
		return ContentRange{}, fmt.Errorf("%w: %q has no total size", ErrInvalidRange, header)
	}
	startPart, endPart, ok := strings.Cut(rangePart, "-")
	if !ok {
		return ContentRange{}, fmt.Errorf("%w: %q has no byte range", ErrInvalidRange, header)
	}

	start, err := strconv.ParseInt(startPart, 10, 64)
	if err != nil {
		return ContentRange{}, fmt.Errorf("%w: bad range start in %q", ErrInvalidRange, header)
	}
	end, err := strconv.ParseInt(endPart, 10, 64)
	if err != nil {
		return ContentRange{}, fmt.Errorf("%w: bad range end in %q", ErrInvalidRange, header)
	}
	total, err := strconv.ParseInt(totalPart, 10, 64)
	if err != nil {
		return ContentRange{}, fmt.Errorf("%w: bad total size in %q", ErrInvalidRange, header)
	}

	parsed := ContentRange{Start: start, End: end, Total: total}
	if start < 0 || end < start || total <= 0 || end >= total {
		return ContentRange{}, fmt.Errorf("%w: %q is not a satisfiable range", ErrInvalidRange, header)
	}
	return parsed, nil
}
