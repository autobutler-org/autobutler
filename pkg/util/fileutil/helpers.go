package fileutil

import "fmt"

// notFound marks an error as a 404 for the handler.
func notFound(err error) error { return &NotFoundError{Err: err} }

// notFoundf marks a formatted message as a 404 for the handler.
func notFoundf(format string, args ...any) error {
	return &NotFoundError{Err: fmt.Errorf(format, args...)}
}
