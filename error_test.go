package rest

import (
	"errors"
	"io/fs"
	"os"
	"testing"
)

// TestErrorUnwrapping tests the Unwrap method of the Error type
func TestErrorUnwrapping(t *testing.T) {
	// Create test error cases with different HTTP status codes
	testCases := []struct {
		name        string
		statusCode  int
		errorMsg    string
		expectedErr error
	}{
		{"Permission Denied", 403, "Permission denied", os.ErrPermission},
		{"Not Found", 404, "Not found", fs.ErrNotExist},
		{"Server Error", 500, "Internal server error", nil},
		{"Custom Error", 400, "Bad request", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a Response with the test case's status code
			resp := &Response{
				Result: "error",
				Error:  tc.errorMsg,
				Code:   tc.statusCode,
			}

			// Create an Error that wraps the Response
			err := &Error{Response: resp}

			// Test the Error() method
			errStr := err.Error()
			expected := "[rest] error from server: " + tc.errorMsg
			if errStr != expected {
				t.Errorf("Error() = %q, want %q", errStr, expected)
			}

			// Test error unwrapping to standard errors
			if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
				t.Errorf("errors.Is(%v, %v) = false, want true", err, tc.expectedErr)
			}
		})
	}

	// Test error with parent
	resp := &Response{
		Result: "error",
		Error:  "wrapped error",
		Code:   500,
	}
	parentErr := errors.New("parent error")
	err := &Error{
		Response: resp,
		parent:   parentErr,
	}

	// Should unwrap to parent, not to the code-based error
	if !errors.Is(err, parentErr) {
		t.Errorf("error with parent did not unwrap to parent")
	}
}

// TestHttpError tests the HttpError type
func TestHttpError(t *testing.T) {
	// Create an HttpError
	body := []byte("Gateway Timeout")
	parentErr := errors.New("connection timeout")
	httpErr := &HttpError{
		Code: 504,
		Body: body,
		e:    parentErr,
	}

	// Test Error() method
	errorMsg := httpErr.Error()
	expectedMsg := "HTTP Error 504: Gateway Timeout"
	if errorMsg != expectedMsg {
		t.Errorf("HttpError.Error() = %q, want %q", errorMsg, expectedMsg)
	}

	// Test Unwrap method
	if !errors.Is(httpErr, parentErr) {
		t.Errorf("errors.Is(httpErr, parentErr) = false, want true")
	}
}
