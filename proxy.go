package rest

import (
	"net/http"
	"net/http/httputil"
)

var (
	SystemProxy = &httputil.ReverseProxy{
		Director:  systemProxyDirector,
		Transport: RestHttpClient.Transport,
	}
)

func systemProxyDirector(req *http.Request) {
	req.URL.Scheme = "https" // ?
	req.URL.Host = "www.atonline.com"
	req.Host = "www.atonline.com"
	req.Header.Set("Host", "www.atonline.com")
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
