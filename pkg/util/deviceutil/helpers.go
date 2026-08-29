package deviceutil

import "fmt"

// invalid marks an error as a 400 for the handler.
func invalid(err error) error { return &InvalidRequestError{Err: err} }

// invalidf marks a formatted message as a 400 for the handler.
func invalidf(format string, args ...any) error {
	return &InvalidRequestError{Err: fmt.Errorf(format, args...)}
}

// unauthorized marks an error as a 401 for the handler.
func unauthorized(err error) error { return &UnauthorizedError{Err: err} }
