package rest

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/KarpelesLab/pjson"
	"github.com/KarpelesLab/typutil"
)

type Param map[string]any

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

func (r *Response) ReadValue(ctx context.Context) (any, error) {
	return r.Value()
}

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

// Raw is implemented as r.Value() for compatibility
func (r *Response) Raw() (any, error) {
	return r.Value()
}

// FullRaw fetches the whole raw object including "Response" own data
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

func (r *Response) Apply(v any) error {
	return pjson.Unmarshal(r.Data, v)
}

func (r *Response) ApplyContext(ctx context.Context, v any) error {
	return pjson.UnmarshalContext(ctx, r.Data, v)
}

func (r *Response) Value() (any, error) {
	r.dataParse.Do(r.ParseData)
	return r.dataParsed, r.dataError
}

func (r *Response) ValueContext(ctx context.Context) (any, error) {
	r.dataParse.Do(r.ParseData)
	return r.dataParsed, r.dataError
}

func (r *Response) ParseData() {
	r.dataError = pjson.Unmarshal(r.Data, &r.dataParsed)
}

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
