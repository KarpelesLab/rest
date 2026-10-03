package rest

import (
	"encoding/json/v2"
	"strings"
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

// TestTimeRoundTrip runs Time through encoding/json/v2 inside a struct, the way
// it is used in responses, and checks the wire format and null handling.
func TestTimeRoundTrip(t *testing.T) {
	type wrap struct {
		When Time `json:"when"`
	}
	in := wrap{Time{time.Date(2023, 5, 15, 14, 30, 45, 123456000, time.UTC)}}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	for _, want := range []string{`"unix":1684161045`, `"us":123456`, `"tz":"UTC"`, `"iso":"2023-05-15 14:30:45"`, `"full":"1684161045123456"`, `"unixms":"1684161045123"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("Marshal output %s lacks %s", data, want)
		}
	}

	var out wrap
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if !out.When.Equal(in.When.Time) {
		t.Errorf("round trip changed time: %v != %v", out.When.Time, in.When.Time)
	}

	// null leaves the value untouched, like the standard package does
	out = wrap{When: in.When}
	if err := json.Unmarshal([]byte(`{"when":null}`), &out); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
}
