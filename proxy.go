//go:build !wasm

package rest

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	SystemProxy = &httputil.ReverseProxy{
		Director:  systemProxyDirector,
		Transport: RestHttpClient.Transport,
	}
)

func systemProxyDirector(req *http.Request) {
	if bk, ok := req.Context().Value(BackendURL).(*url.URL); ok && bk != nil {
		req.URL.Scheme = bk.Scheme
		req.URL.Host = bk.Host
	} else {
		req.URL.Scheme = Scheme
		req.URL.Host = Host
	}
	//req.Host = Host
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("Sec-Rest-Http", "true")
	req.Header.Del("Accept-Encoding")

	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
	if _, ok := req.Header["Cookie"]; ok {
		req.Header.Del("Cookie")
	}
	// let context alter request as needed
	req.Context().Value(req)
}
