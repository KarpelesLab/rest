// Package rest provides a client for interacting with RESTful API services.
package rest

import (
	"net/http"
	"time"
)

// RestHttpTransport is the configured HTTP transport used for all REST API requests.
// It's optimized with connection pooling and appropriate timeouts.
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

// RestHttpClient is the default HTTP client used for all REST API requests.
// It uses RestHttpTransport and has a 5-minute overall timeout.
var RestHttpClient = &http.Client{
	Transport: RestHttpTransport,
	Timeout:   300 * time.Second,
}
