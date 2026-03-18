package serverutil

import "fmt"

// HttpError is an error type that carries an HTTP status code.
// When returned from a service function and detected by WrapApiRoute,
// its StatusCode will be used instead of the default 500.
type HttpError struct {
	StatusCode int
	Message    string
}

func (e *HttpError) Error() string {
	return e.Message
}

// NewHttpError creates an HttpError with the given status code and message.
func NewHttpError(statusCode int, message string) *HttpError {
	return &HttpError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// NewHttpErrorf creates an HttpError with the given status code and a formatted message.
func NewHttpErrorf(statusCode int, format string, args ...any) *HttpError {
	return &HttpError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf(format, args...),
	}
}
