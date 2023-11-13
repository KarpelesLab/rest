package rest

import (
	"net/http"
	"time"
)

var RestHttpTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   50,
	MaxConnsPerHost:       200,
	IdleConnTimeout:       90 * time.Second,
	ResponseHeaderTimeout: 90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 5 * time.Second,
}

var RestHttpClient = &http.Client{
	Transport: RestHttpTransport,
	Timeout:   120 * time.Second,
}
