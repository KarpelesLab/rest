package rest

import (
	"context"
	"testing"
)

// TestResponse tests various methods of the Response struct
func TestResponse(t *testing.T) {
	// Create a test Response object
	jsonData := []byte(`{"name":"test","value":42,"nested":{"key":"value"},"array":[1,2,3]}`)
	resp := &Response{
		Result: "success",
		Data:   jsonData,
		Error:  "",
		Code:   0,
		Extra:  "extra info",
		Token:  "test-token",
	}

	// Test ReadValue method
	value, err := resp.ReadValue(context.Background())
	if err != nil {
		t.Errorf("ReadValue failed: %s", err)
	}
	if value == nil {
		t.Errorf("ReadValue returned nil value")
	}

	// Test OffsetGet with metadata fields
	// @-prefixed keys
	ctx := context.Background()
	errorVal, err := resp.OffsetGet(ctx, "@error")
	if err != nil || errorVal != "" {
		t.Errorf("OffsetGet(@error) = %v, %v, want \"\", nil", errorVal, err)
	}

	codeVal, err := resp.OffsetGet(ctx, "@code")
	if err != nil || codeVal != 0 {
		t.Errorf("OffsetGet(@code) = %v, %v, want 0, nil", codeVal, err)
	}

	extraVal, err := resp.OffsetGet(ctx, "@extra")
	if err != nil || extraVal != "extra info" {
		t.Errorf("OffsetGet(@extra) = %v, %v, want \"extra info\", nil", extraVal, err)
	}

	tokenVal, err := resp.OffsetGet(ctx, "@token")
	if err != nil || tokenVal != "test-token" {
		t.Errorf("OffsetGet(@token) = %v, %v, want \"test-token\", nil", tokenVal, err)
	}

	// Test data fields via OffsetGet
	nameVal, err := resp.OffsetGet(ctx, "name")
	if err != nil || nameVal != "test" {
		t.Errorf("OffsetGet(name) = %v, %v, want \"test\", nil", nameVal, err)
	}

	// Test Raw method
	raw, err := resp.Raw()
	if err != nil {
		t.Errorf("Raw failed: %s", err)
	}
	if raw == nil {
		t.Errorf("Raw returned nil value")
	}

	// Test FullRaw method
	fullRaw, err := resp.FullRaw()
	if err != nil {
		t.Errorf("FullRaw failed: %s", err)
	}
	if fullRaw == nil {
		t.Errorf("FullRaw returned nil value")
	}
	if fullRaw["result"] != "success" {
		t.Errorf("FullRaw result = %v, want %q", fullRaw["result"], "success")
	}
	if fullRaw["extra"] != "extra info" {
		t.Errorf("FullRaw extra = %v, want %q", fullRaw["extra"], "extra info")
	}

	// Test ApplyContext method
	var target struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	err = resp.ApplyContext(context.Background(), &target)
	if err != nil {
		t.Errorf("ApplyContext failed: %s", err)
	}
	if target.Name != "test" || target.Value != 42 {
		t.Errorf("ApplyContext result = {%s, %d}, want {test, 42}", target.Name, target.Value)
	}

	// Test ValueContext method
	valueCtx, err := resp.ValueContext(context.Background())
	if err != nil {
		t.Errorf("ValueContext failed: %s", err)
	}
	if valueCtx == nil {
		t.Errorf("ValueContext returned nil value")
	}
}

// TestResponsePathAccess tests the Get and GetString methods for path access
func TestResponsePathAccess(t *testing.T) {
	// Create a deeply nested test Response object
	jsonData := []byte(`{
		"level1": {
			"level2": {
				"level3": {
					"string": "nested value",
					"number": 42,
					"bool": true,
					"array": [1, 2, 3]
				}
			},
			"sibling": "sibling value"
		},
		"empty": null
	}`)

	resp := &Response{
		Result: "success",
		Data:   jsonData,
	}

	// Test Get with valid paths
	tests := []struct {
		path string
		want any
	}{
		{"level1/level2/level3/string", "nested value"},
		{"level1/sibling", "sibling value"},
		{"level1/level2/level3/number", float64(42)}, // JSON numbers are floats
		{"level1/level2/level3/bool", true},
		{"level1", map[string]any{}}, // Just check it's a map, content varies
		{"nonexistent", nil},
		{"empty", nil},
		{"", map[string]any{}}, // Empty path returns the root
	}

	for _, tt := range tests {
		got, err := resp.Get(tt.path)
		if err != nil {
			t.Errorf("Get(%q) error = %v", tt.path, err)
			continue
		}

		switch tt.want.(type) {
		case map[string]any:
			// For maps, just check the type
			if _, ok := got.(map[string]any); !ok {
				t.Errorf("Get(%q) = %T, want map[string]any", tt.path, got)
			}
		default:
			// For other types, check the exact value
			if got != tt.want {
				t.Errorf("Get(%q) = %v, want %v", tt.path, got, tt.want)
			}
		}
	}

	// Test GetString with various path types
	stringTests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{"level1/level2/level3/string", "nested value", false},
		{"level1/sibling", "sibling value", false},
		{"level1/level2/level3/number", "", true}, // Not a string
		{"level1/level2/level3/bool", "", true},   // Not a string
		{"nonexistent", "", true},                 // Path doesn't exist
		{"empty", "", true},                       // Null value
	}

	for _, tt := range stringTests {
		got, err := resp.GetString(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("GetString(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("GetString(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
