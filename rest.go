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

	"github.com/MagicalTux/gophp/core/util"
)

type RestParam map[string]interface{}

type RestResponse struct {
	Result string          `json:"result"` // "success" or "error" (or "redirect")
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error"`

	Paging interface{} `json:"paging"`
	Job    interface{} `json:"job"`
	Time   interface{} `json:"time"`
	Access interface{} `json:"access"`

	RedirectUrl  string `json:"redirect_url"`
	RedirectCode int    `json:"redirect_code"`
}

func (r *RestResponse) ReadValue(ctx context.Context) (interface{}, error) {
	var v interface{}
	err := json.Unmarshal(r.Data, &v)
	return v, err
}

func RestJson(ctx context.Context, req, method string, param map[string]interface{}, target interface{}) error {
	res, err := NewRest(ctx, req, method, param)
	if err != nil {
		return err
	}
	err = json.Unmarshal(res.Data, target)
	if err != nil {
		log.Printf("failed to parse json: %s %s", err, res.Data)
	}
	return err
}

func NewRest(ctx context.Context, req, method string, param map[string]interface{}) (*RestResponse, error) {
	// build http request
	r := &http.Request{
		Method: method,
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.atonline.com",
			Path:   "/_special/rest/" + req,
		},
		Header: make(http.Header),
	}

	r.Header.Set("Sec-Rest-Http", "false")

	// add parameters (depending on method)
	switch method {
	case "GET", "HEAD", "OPTIONS":
		// need to pass parameters in GET
		r.URL.RawQuery = util.EncodePhpQuery(param)
	case "POST", "PATCH":
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

	t := time.Now()

	resp, err := RestHttpClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	d := time.Since(t)
	log.Printf("[rest] %s %s => %s", method, req, d)

	//util.CtxPrintf(ctx, "[debug] Response to %s %s: %s", method, req, body)

	result := &RestResponse{}
	err = json.Unmarshal(body, result)
	if err != nil {
		log.Printf("failed to parse json: %s %s", err, body)
		return nil, err
	}

	if result.Result == "redirect" {
		url, err := url.Parse(result.RedirectUrl)
		if err != nil {
			return nil, err
		}
		return nil, RedirectErrorCode(url, result.RedirectCode)
	}

	if result.Result == "error" {
		return nil, fmt.Errorf("[rest] error from server: %s", result.Error)
	}

	return result, nil
}
