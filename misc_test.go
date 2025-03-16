package rest

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"
)

// TestFixedArray tests the fixedArray endpoint which returns a fixed array
func TestFixedArray(t *testing.T) {
	ctx := context.Background()
	
	// Using Apply to unmarshal into a map
	var result map[string]interface{}
	err := Apply(ctx, "Misc/Debug:fixedArray", "GET", nil, &result)
	if err != nil {
		t.Fatalf("failed to call fixedArray: %s", err)
	}
	
	// Verify we got an array with at least one element
	if len(result) == 0 {
		t.Errorf("expected non-empty array, got empty result")
	}
	
	// Test using the generic As method
	res, err := As[map[string]interface{}](ctx, "Misc/Debug:fixedArray", "GET", nil)
	if err != nil {
		t.Fatalf("failed to call fixedArray with As: %s", err)
	}
	
	if len(res) == 0 {
		t.Errorf("expected non-empty array with As method, got empty result")
	}
}

// TestFixedString tests the fixedString endpoint which returns a fixed string
func TestFixedString(t *testing.T) {
	ctx := context.Background()
	
	res, err := Do(ctx, "Misc/Debug:fixedString", "GET", nil)
	if err != nil {
		t.Fatalf("failed to call fixedString: %s", err)
	}
	
	// Get the string value using the Response.Get method
	str, err := res.GetString("")
	if err != nil {
		t.Fatalf("failed to get string from response: %s", err)
	}
	
	if str == "" {
		t.Errorf("expected non-empty string, got empty string")
	}
}

// TestError tests the error endpoint which returns an error response
func TestError(t *testing.T) {
	ctx := context.Background()
	
	_, err := Do(ctx, "Misc/Debug:error", "GET", nil)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	
	// Test error unwrapping
	var restErr *Error
	if !errors.As(err, &restErr) {
		t.Errorf("expected error to be of type *rest.Error, got %T", err)
	}
}

// TestErrorUnwrap tests error handling mechanisms
func TestErrorUnwrap(t *testing.T) {
	ctx := context.Background()
	
	// Let's construct a test *Error object manually to verify the unwrapping works
	resp403 := &Response{
		Result: "error",
		Error:  "Permission denied",
		Code:   403,
	}
	
	err403 := &Error{Response: resp403}
	
	// Test the error unwrapping logic
	if !errors.Is(err403, os.ErrPermission) {
		t.Errorf("Error with code 403 should unwrap to os.ErrPermission")
	}
	
	resp404 := &Response{
		Result: "error",
		Error:  "Not found",
		Code:   404,
	}
	
	err404 := &Error{Response: resp404}
	
	// Test the error unwrapping logic
	if !errors.Is(err404, fs.ErrNotExist) {
		t.Errorf("Error with code 404 should unwrap to fs.ErrNotExist")
	}
	
	// Test with the fieldError endpoint (don't rely on specific codes as they may change)
	_, err := Do(ctx, "Misc/Debug:fieldError", "GET", Param{"i": 42})
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	
	// Verify it's a *Error type
	var restErr *Error
	if !errors.As(err, &restErr) {
		t.Errorf("expected error to be of type *rest.Error, got %T", err)
	}
}

// TestRedirect tests the testRedirect endpoint which returns a redirect response
func TestRedirect(t *testing.T) {
	ctx := context.Background()
	
	_, err := Do(ctx, "Misc/Debug:testRedirect", "GET", nil)
	if err == nil {
		t.Fatalf("expected redirect error but got nil")
	}
	
	// The testRedirect endpoint seems to redirect to www.perdu.com
	// Just check that we get a redirect error without hardcoding the exact URL
	errMsg := err.Error()
	if errMsg == "" || (!contains(errMsg, "redirect") && !contains(errMsg, "Redirect")) {
		t.Errorf("expected redirect error message, got: %s", errMsg)
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestArgument tests the argument endpoint with various input parameters
func TestArgument(t *testing.T) {
	ctx := context.Background()
	
	testValue := "hello world"
	
	// Test with the required 'input' parameter
	res, err := Do(ctx, "Misc/Debug:argument", "GET", Param{"input": testValue})
	if err != nil {
		t.Fatalf("failed to call argument endpoint: %s", err)
	}
	
	// The endpoint should return our input value
	returnedValue, err := res.GetString("input")
	if err != nil {
		t.Fatalf("failed to get input value from response: %s", err)
	}
	
	if returnedValue != testValue {
		t.Errorf("expected returned value '%s', got '%s'", testValue, returnedValue)
	}
}

// TestArgString tests the argString endpoint which accepts a string parameter
func TestArgString(t *testing.T) {
	ctx := context.Background()
	
	testValue := "test string"
	
	// Using the generic As method to unmarshal directly into a map
	res, err := As[map[string]interface{}](ctx, "Misc/Debug:argString", "GET", Param{"input_string": testValue})
	if err != nil {
		t.Fatalf("failed to call argString endpoint: %s", err)
	}
	
	// The endpoint should echo the input_string in the response
	if v, ok := res["input_string"]; !ok || v != testValue {
		t.Errorf("expected returned input_string '%s', got '%v'", testValue, v)
	}
}

// TestResponseAs tests the ResponseAs generic function
func TestResponseAs(t *testing.T) {
	// Create a sample Response with JSON data
	jsonData := []byte(`{"name":"test","value":42,"items":["one","two","three"]}`)
	resp := &Response{
		Result: "success",
		Data:   jsonData,
	}
	
	// Define a struct that matches our expected data structure
	type TestData struct {
		Name  string   `json:"name"`
		Value int      `json:"value"`
		Items []string `json:"items"`
	}
	
	// Use ResponseAs to unmarshal the data
	data, err := ResponseAs[TestData](resp)
	if err != nil {
		t.Fatalf("ResponseAs failed: %s", err)
	}
	
	// Verify the data was correctly unmarshaled
	if data.Name != "test" {
		t.Errorf("expected Name='test', got '%s'", data.Name)
	}
	
	if data.Value != 42 {
		t.Errorf("expected Value=42, got %d", data.Value)
	}
	
	if len(data.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(data.Items))
	} else if data.Items[0] != "one" || data.Items[1] != "two" || data.Items[2] != "three" {
		t.Errorf("items not correctly unmarshaled: %v", data.Items)
	}
}