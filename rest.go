package rest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/KarpelesLab/pjson"
	"github.com/KarpelesLab/webutil"
)

var (
	Debug  = false
	Scheme = "https"
	Host   = "www.atonline.com"
)

func Apply(ctx context.Context, req, method string, param any, target any) error {
	res, err := Do(ctx, req, method, param)
	if err != nil {
		return err
	}
	err = pjson.UnmarshalContext(ctx, res.Data, target)
	if Debug && err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("failed to parse json: %s\n%s", err, res.Data), "event", "rest:not_json")
	}
	return err
}

func Do(ctx context.Context, req, method string, param any) (*Response, error) {
	var backend *url.URL
	if bk, ok := ctx.Value(BackendURL).(*url.URL); ok && bk != nil {
		backend = bk
	} else {
		backend = &url.URL{Scheme: Scheme, Host: Host}
	}
	// build http request
	r := &http.Request{
		Method: method,
		URL: &url.URL{
			Scheme: backend.Scheme,
			Host:   backend.Host,
			Path:   "/_special/rest/" + req,
		},
		Header: make(http.Header),
	}

	r.Header.Set("Sec-Rest-Http", "false")

	// add parameters (depending on method)
	switch method {
	case "GET", "HEAD", "OPTIONS":
		// need to pass parameters in GET
		data, err := pjson.MarshalContext(ctx, param)
		if err != nil {
			return nil, err
		}
		r.URL.RawQuery = "_=" + url.QueryEscape(string(data))
	case "PUT", "POST", "PATCH":
		data, err := pjson.MarshalContext(ctx, param)
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
	err = pjson.UnmarshalContext(ctx, body, result)
	if err != nil {
		if Debug {
			slog.ErrorContext(ctx, fmt.Sprintf("failed to parse json: %s\n%s", err, body), "event", "rest:not_json")
		}
		return nil, err
	}

	if token != nil && result.Token == "invalid_request_token" && result.Extra == "token_expired" {
		// token has expired, renew token & re-run process
		if Debug {
			slog.DebugContext(ctx, "Token has expired, requesting renew", "event", "rest:token_renew")
		}
		if err := token.renew(ctx); err != nil {
			// error
			if Debug {
				slog.ErrorContext(ctx, fmt.Sprintf("failed to renew token: %s", err), "event", "rest:token_renew_fail")
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

		err = pjson.UnmarshalContext(ctx, body, result)
		if err != nil {
			if Debug {
				slog.ErrorContext(ctx, fmt.Sprintf("failed to parse json: %s\n%s", err, body), "event", "rest:not_json")
			}
			return nil, err
		}
	}

	if Debug {
		d := time.Since(t)
		slog.DebugContext(ctx, fmt.Sprintf("[rest] %s %s => %s", method, req, d), "event", "rest:debug_query", "rest:method", method, "rest:request", req, "rest:duration", d)
	}

	if result.Result == "redirect" {
		if result.Exception == "Exception\\Login" {
			return nil, ErrLoginRequired
		}
		url, err := url.Parse(result.RedirectUrl)
		if err != nil {
			return nil, err
		}
		return nil, webutil.RedirectErrorCode(url, result.RedirectCode)
	}

	if result.Result == "error" {
		return nil, &Error{Response: result}
	}

	return result, nil
}
