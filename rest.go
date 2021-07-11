package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

var (
	Debug = false
	Host  = "www.atonline.com"
)

type Param map[string]interface{}

type Response struct {
	Result string          `json:"result"` // "success" or "error" (or "redirect")
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	Extra  string          `json:"extra,omitempty"`
	Token  string          `json:"token,omitempty"`

	Paging interface{} `json:"paging,omitempty"`
	Job    interface{} `json:"job,omitempty"`
	Time   interface{} `json:"time,omitempty"`
	Access interface{} `json:"access,omitempty"`

	RedirectUrl  string `json:"redirect_url,omitempty"`
	RedirectCode int    `json:"redirect_code,omitempty"`
}

func (r *Response) ReadValue(ctx context.Context) (interface{}, error) {
	var v interface{}
	err := json.Unmarshal(r.Data, &v)
	return v, err
}

func (r *Response) Apply(v interface{}) error {
	return json.Unmarshal(r.Data, v)
}

func Apply(ctx context.Context, req, method string, param Param, target interface{}) error {
	res, err := Do(ctx, req, method, param)
	if err != nil {
		return err
	}
	err = json.Unmarshal(res.Data, target)
	if Debug && err != nil {
		log.Printf("failed to parse json: %s %s", err, res.Data)
	}
	return err
}

func Do(ctx context.Context, req, method string, param Param) (*Response, error) {
	// build http request
	r := &http.Request{
		Method: method,
		URL: &url.URL{
			Scheme: "https",
			Host:   Host,
			Path:   "/_special/rest/" + req,
		},
		Header: make(http.Header),
	}

	r.Header.Set("Sec-Rest-Http", "false")

	// add parameters (depending on method)
	switch method {
	case "GET", "HEAD", "OPTIONS":
		// need to pass parameters in GET
		data, err := json.Marshal(param)
		if err != nil {
			return nil, err
		}
		r.URL.RawQuery = "_=" + url.QueryEscape(string(data))
	case "PUT", "POST", "PATCH":
		data, err := json.Marshal(param)
		if err != nil {
			return nil, err
		}
		buf := bytes.NewReader(data)
		r.Body = ioutil.NopCloser(buf)
		r.ContentLength = int64(len(data))
		r.GetBody = func() (io.ReadCloser, error) {
			reader := bytes.NewReader(data)
			return ioutil.NopCloser(reader), nil
		}
		r.Header.Set("Content-Type", "application/json")
	case "DELETE":
		// nothing
	default:
		return nil, fmt.Errorf("invalid request method %s", method)
	}

	// final configuration
	ctx.Value(r)

	// check for rest token
	var token *Token
	if t, ok := ctx.Value(tokenValue(0)).(*Token); t != nil && ok {
		// set token & authorization header
		token = t
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	}

	t := time.Now()

	resp, err := RestHttpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to run rest query: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//log.Printf(ctx, "[rest] Response to %s %s: %s", method, req, body)

	result := &Response{}
	err = json.Unmarshal(body, result)
	if err != nil {
		if Debug {
			log.Printf("[rest] failed to parse json: %s %s", err, body)
		}
		return nil, err
	}

	if token != nil && result.Token == "invalid_request_token" && result.Extra == "token_expired" {
		// token has expired, renew token & re-run process
		if Debug {
			log.Printf("[rest] Token has expired, requesting renew")
		}
		if err := token.renew(ctx); err != nil {
			// error
			if Debug {
				log.Printf("[rest] failed to renew token: %s", err)
			}
			return nil, err
		}

		// re-run query
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
		resp, err := RestHttpClient.Do(r)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(body, result)
		if err != nil {
			if Debug {
				log.Printf("[rest] failed to parse json: %s %s", err, body)
			}
			return nil, err
		}
	}

	if Debug {
		d := time.Since(t)
		log.Printf("[rest] %s %s => %s", method, req, d)
	}

	if result.Result == "redirect" {
		url, err := url.Parse(result.RedirectUrl)
		if err != nil {
			return nil, err
		}
		return nil, RedirectErrorCode(url, result.RedirectCode)
	}

	if result.Result == "error" {
		return nil, &Error{result}
	}

	return result, nil
}
