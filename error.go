package rest

import "fmt"

type Error struct {
	Response *Response
}

func (r *Error) Error() string {
	return fmt.Sprintf("[rest] error from server: %s", r.Response.Error)
}
