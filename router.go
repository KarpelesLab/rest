//go:build !wasm

package rest

import "net/http"

type RouterType struct {
}

var Router *RouterType = &RouterType{}

func (h *RouterType) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// fallback to PHP, add prefix for rest
	req.URL.Path = "/_special/rest" + req.URL.Path
	SystemProxy.ServeHTTP(w, req)
}
