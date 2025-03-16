// Package rest provides a client for interacting with RESTful API services.
package rest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// ErrLoginRequired is returned when an API endpoint requires authentication
// but no valid token was provided.
var ErrLoginRequired = errors.New("login required")

// Error represents an error returned by a REST API endpoint.
// It wraps the Response object and provides standard error interfaces.
type Error struct {
	Response *Response
	parent   error
}

// Error returns a string representation of the REST API error.
func (r *Error) Error() string {
	return fmt.Sprintf("[rest] error from server: %s", r.Response.Error)
}

// Unwrap implements the errors.Unwrapper interface to allow error checking with errors.Is.
// It maps REST API errors to standard Go errors where applicable (e.g., 403 to os.ErrPermission).
func (r *Error) Unwrap() error {
	if r.parent != nil {
		return r.parent
	}
	// check for various type of errors
	switch r.Response.Code {
	case 403:
		return os.ErrPermission
	case 404:
		return fs.ErrNotExist
	default:
		return nil
	}
}

// HttpError represents an HTTP transport error that occurred during a REST API request.
// It captures HTTP status codes and response bodies for debugging.
type HttpError struct {
	Code int
	Body []byte
	e    error // unwrap error
}

// Error returns a string representation of the HTTP error.
func (e *HttpError) Error() string {
	return fmt.Sprintf("HTTP Error %d: %s", e.Code, e.Body)
}

// Unwrap implements the errors.Unwrapper interface to allow error checking with errors.Is.
// It returns the underlying error, if any.
func (e *HttpError) Unwrap() error {
	return e.e
}
