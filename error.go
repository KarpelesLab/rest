package rest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

var ErrLoginRequired = errors.New("login required")

type Error struct {
	Response *Response
	parent   error
}

func (r *Error) Error() string {
	return fmt.Sprintf("[rest] error from server: %s", r.Response.Error)
}

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
