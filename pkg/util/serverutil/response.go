package serverutil

import (
	"net/http"
)

type Response struct {
	StatusCode  int
	Data        any
	Error       error
	ContentType ContentType
}

func Accepted() *Response {
	return NewResponse().WithStatusCode(http.StatusAccepted)
}

func Ok() *Response {
	return NewResponse().WithStatusCode(http.StatusOK)
}

func BadRequest(err error) *Response {
	return NewResponse().WithStatusCode(http.StatusBadRequest).WithError(err)
}

func Unauthorized(err error) *Response {
	return NewResponse().WithStatusCode(http.StatusUnauthorized).WithError(err)
}

func InternalServerError(err error) *Response {
	return NewResponse().WithStatusCode(http.StatusInternalServerError).WithError(err)
}

// Conflict reports a request that cannot proceed against the current state —
// a chunk offered at the wrong offset of an upload session, for instance
// (#1629). Callers that want the client to resync set the headers carrying the
// true state before returning this.
func Conflict(err error) *Response {
	return NewResponse().WithStatusCode(http.StatusConflict).WithError(err)
}

func NotFound(err error) *Response {
	return NewResponse().WithStatusCode(http.StatusNotFound).WithError(err)
}

func ServiceUnavailable(err error) *Response {
	return NewResponse().WithStatusCode(http.StatusServiceUnavailable).WithError(err)
}

func NewResponse() *Response {
	return (&Response{}).WithContentType(ContentTypeJSON).WithStatusCode(http.StatusOK).WithData(nil).WithError(nil)
}

func (r *Response) WithContentType(contentType ContentType) *Response {
	r.ContentType = contentType
	return r
}

func (r *Response) WithStatusCode(statusCode int) *Response {
	r.StatusCode = statusCode
	return r
}

func (r *Response) WithData(data any) *Response {
	r.Data = data
	return r
}

func (r *Response) WithError(err error) *Response {
	r.Error = err
	return r
}
