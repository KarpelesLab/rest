package rest

import (
	"context"
	"testing"
	"time"
)

// TestTimeUnmarshalJSON tests the UnmarshalJSON method of Time
func TestTimeUnmarshalJSON(t *testing.T) {
	// Create test cases with various JSON formats
	tests := []struct {
		name string
		json []byte
	}{
		{"unix timestamp", []byte(`{"unix":1684162245,"us":0}`)},
		{"null", []byte(`null`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tm Time
			err := tm.UnmarshalJSON(tt.json)
			if err != nil {
				t.Errorf("UnmarshalJSON(%s) error = %v", tt.json, err)
			}
		})
	}
}

// TestTimeMarshalJSON tests the MarshalJSON method of Time
func TestTimeMarshalJSON(t *testing.T) {
	// Create a time value
	tm := Time{time.Date(2023, 5, 15, 14, 30, 45, 0, time.UTC)}

	// Test marshaling
	data, err := tm.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON error = %v", err)
	}

	if len(data) == 0 {
		t.Errorf("MarshalJSON returned empty data")
	}
}

// TestTimeContextJSON tests the context-based JSON methods
func TestTimeContextJSON(t *testing.T) {
	// Create a time
	tm := Time{time.Date(2023, 5, 15, 14, 30, 45, 0, time.UTC)}
	ctx := context.Background()

	// Test MarshalContextJSON
	data, err := tm.MarshalContextJSON(ctx)
	if err != nil {
		t.Errorf("MarshalContextJSON error = %v", err)
	}

	if len(data) == 0 {
		t.Errorf("MarshalContextJSON returned empty data")
	}

	// Test UnmarshalContextJSON
	var tm2 Time
	err = tm2.UnmarshalContextJSON(ctx, data)
	if err != nil {
		t.Errorf("UnmarshalContextJSON error = %v", err)
	}

	// Also test null handling
	err = tm2.UnmarshalContextJSON(ctx, []byte(`null`))
	if err != nil {
		t.Errorf("UnmarshalContextJSON(null) error = %v", err)
	}
}
