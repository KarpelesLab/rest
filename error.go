package rest

import (
	"fmt"
	"io/fs"
	"os"
)

type Error struct {
	Response *Response
}

func (r *Error) Error() string {
	return fmt.Sprintf("[rest] error from server: %s", r.Response.Error)
}

func (r *Error) Unwrap() error {
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
