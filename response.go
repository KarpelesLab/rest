package rest

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/KarpelesLab/pjson"
	"github.com/KarpelesLab/typutil"
)

// Param is a convenience type for parameters passed to REST API requests.
type Param map[string]any

// Response represents a REST API response with standard fields.
// It handles different result types and provides methods to access response data.
type Response struct {
	Result string           `json:"result"` // "success" or "error" (or "redirect")
	Data   pjson.RawMessage `json:"data,omitempty"`
	Error  string           `json:"error,omitempty"`
	Code   int              `json:"code,omitempty"` // for errors
	Extra  string           `json:"extra,omitempty"`
	Token  string           `json:"token,omitempty"`

	Paging any `json:"paging,omitempty"`
	Job    any `json:"job,omitempty"`
	Time   any `json:"time,omitempty"`
	Access any `json:"access,omitempty"`

	Exception    string `json:"exception,omitempty"`
	RedirectUrl  string `json:"redirect_url,omitempty"`
	RedirectCode int    `json:"redirect_code,omitempty"`

	dataParsed any
	dataError  error
	dataParse  sync.Once
}

// ReadValue returns the parsed data from the response.
// It's an alias for Value() that satisfies interfaces requiring a context parameter.
func (r *Response) ReadValue(ctx context.Context) (any, error) {
	return r.Value()
}

// OffsetGet implements the typutil.Getter interface for Response objects.
// It allows accessing response fields by key, with special handling for metadata keys
// prefixed with '@' (e.g., @error, @code).
func (r *Response) OffsetGet(ctx context.Context, key string) (any, error) {
	if strings.HasPrefix(key, "@") {
		switch key[1:] {
		case "error":
			return r.Error, nil
		case "code":
			return r.Code, nil
		case "extra":
			return r.Extra, nil
		case "token":
			return r.Token, nil
		case "paging":
			return r.Paging, nil
		case "job":
			return r.Job, nil
		case "time":
			return r.Time, nil
		case "access":
			return r.Access, nil
		case "exception":
			return r.Exception, nil
		}
	}

	// return value
	return r.Get(key)
}

// Raw returns the parsed data from the response.
// It's implemented as r.Value() for compatibility with older code.
func (r *Response) Raw() (any, error) {
	return r.Value()
}

// FullRaw returns the complete response as a map, including both the data payload
// and all metadata fields (result, error, code, etc.).
func (r *Response) FullRaw() (map[string]any, error) {
	data, err := r.Value()
	if err != nil {
		return nil, err
	}
	resp := map[string]any{"result": r.Result, "data": data}
	if r.Error != "" {
		resp["error"] = r.Error
	}
	if r.Code != 0 {
		resp["code"] = r.Code
	}
	if r.Extra != "" {
		resp["extra"] = r.Extra
	}
	if r.Token != "" {
		resp["token"] = r.Token
	}
	if r.Paging != nil {
		resp["paging"] = r.Paging
	}
	if r.Job != nil {
		resp["job"] = r.Job
	}
	if r.Time != nil {
		resp["time"] = r.Time
	}
	if r.Access != nil {
		resp["access"] = r.Access
	}
	if r.Exception != "" {
		resp["exception"] = r.Exception
	}
	if r.RedirectUrl != "" {
		resp["redirect_url"] = r.RedirectUrl
	}
	if r.RedirectCode != 0 {
		resp["redirect_code"] = r.RedirectCode
	}

	return resp, nil
}

// Apply unmarshals the response data into the provided value.
//
// Parameters:
//
// - v: The target object to unmarshal into
//
// Returns: an error if unmarshaling fails
func (r *Response) Apply(v any) error {
	return pjson.Unmarshal(r.Data, v)
}

// ResponseAs is a generic helper that unmarshals a response into type T.
//
// Parameters:
//
// - r: The Response object containing data to unmarshal
//
// Returns: the unmarshaled object of type T and any error encountered
func ResponseAs[T any](r *Response) (T, error) {
	var target T
	err := r.Apply(&target)
	return target, err
}

// ApplyContext unmarshals the response data into the provided value using a context.
//
// Parameters:
//
// - ctx: Context for unmarshaling
// - v: The target object to unmarshal into
//
// Returns: an error if unmarshaling fails
func (r *Response) ApplyContext(ctx context.Context, v any) error {
	return pjson.UnmarshalContext(ctx, r.Data, v)
}

// Value returns the parsed data from the response.
// It lazily parses the JSON data on first access and caches the result.
//
// Returns: the parsed data and any error encountered during parsing
func (r *Response) Value() (any, error) {
	r.dataParse.Do(r.ParseData)
	return r.dataParsed, r.dataError
}

// ValueContext returns the parsed data from the response, similar to Value().
// It's provided for interface compatibility with methods requiring a context.
//
// Parameters:
//
// - ctx: Context (not used internally but provided for interface compatibility)
//
// Returns: the parsed data and any error encountered during parsing
func (r *Response) ValueContext(ctx context.Context) (any, error) {
	r.dataParse.Do(r.ParseData)
	return r.dataParsed, r.dataError
}

// ParseData parses the JSON data in the response.
// This is called automatically by Value() and ValueContext() methods.
func (r *Response) ParseData() {
	r.dataError = pjson.Unmarshal(r.Data, &r.dataParsed)
}

// Get retrieves a value from the response data by a slash-separated path.
// For example, "user/name" would access the "name" field inside the "user" object.
//
// Parameters:
//
// - v: Slash-separated path to the requested value
//
// Returns: the value at the specified path and any error encountered
func (r *Response) Get(v string) (any, error) {
	va := strings.Split(v, "/")
	cur, err := r.Value()
	if err != nil {
		return nil, err
	}

	for _, sub := range va {
		if sub == "" {
			continue
		}
		// we assume each sub will be an index in cur as a map
		cur, err = typutil.OffsetGet(context.Background(), cur, sub)
		if err != nil {
			return cur, err
		}
		if cur == nil {
			return nil, nil
		}
	}
	return cur, nil
}

// GetString retrieves a string value from the response data by a slash-separated path.
// This is a convenience method that calls Get() and converts the result to a string.
//
// Parameters:
//
// - v: Slash-separated path to the requested string value
//
// Returns: the string value at the specified path and any error encountered
func (r *Response) GetString(v string) (string, error) {
	res, err := r.Get(v)
	if err != nil {
		return "", err
	}
	str, ok := res.(string)
	if !ok {
		return fmt.Sprintf("%v", str), fmt.Errorf("unexpected type %T for string %s", res, v)
	}
	return str, nil
}
