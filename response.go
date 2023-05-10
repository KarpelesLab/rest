package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

type Param map[string]any

type Response struct {
	Result string          `json:"result"` // "success" or "error" (or "redirect")
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	Code   int             `json:"code,omitempty"` // for errors
	Extra  string          `json:"extra,omitempty"`
	Token  string          `json:"token,omitempty"`

	Paging any `json:"paging,omitempty"`
	Job    any `json:"job,omitempty"`
	Time   any `json:"time,omitempty"`
	Access any `json:"access,omitempty"`

	RedirectUrl  string `json:"redirect_url,omitempty"`
	RedirectCode int    `json:"redirect_code,omitempty"`

	dataParsed any
	dataError  error
	dataParse  sync.Once
}

func (r *Response) ReadValue(ctx context.Context) (any, error) {
	return r.Value()
}

func (r *Response) Apply(v any) error {
	return json.Unmarshal(r.Data, v)
}

func (r *Response) Value() (any, error) {
	r.dataParse.Do(func() {
		r.dataError = json.Unmarshal(r.Data, &r.dataParsed)
	})
	return r.dataParsed, r.dataError
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
		curV, ok := cur.(map[string]any)
		if !ok {
			return nil, fs.ErrNotExist
		}
		cur, ok = curV[sub]
		if !ok {
			return nil, fs.ErrNotExist
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
