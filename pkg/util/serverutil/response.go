package serverutil

import (
	"bytes"
	"context"
	"net/http"

	"github.com/a-h/templ"
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

func (r *Response) WithComponent(component templ.Component) *Response {
	var buf bytes.Buffer
	component.Render(context.Background(), &buf)
	r.Data = buf.String()
	return r
}
