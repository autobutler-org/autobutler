package api

import "net/http"

type ContentType string

const (
	ContentTypeHTML ContentType = "text/html"
	ContentTypeJSON ContentType = "application/json"
)

type Response struct {
	StatusCode  int
	Data        any
	Error       error
	ContentType ContentType
}

func Ok() *Response {
	return NewResponse().WithContentType(ContentTypeHTML).WithStatusCode(http.StatusOK)
}

func NewResponse() *Response {
	return (&Response{}).WithContentType(ContentTypeHTML).WithStatusCode(http.StatusOK).WithData(nil).WithError(nil)
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
