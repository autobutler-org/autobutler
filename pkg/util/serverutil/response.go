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

func InternalServerError(err error) *Response {
	return NewResponse().WithStatusCode(http.StatusInternalServerError).WithError(err)
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
