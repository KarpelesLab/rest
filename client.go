package rest

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/TrisTech/fleet"
)

var RestHttpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialTLS:               restDialTLS,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	Timeout: 120 * time.Second,
}

func restDialTLS(network, addr string) (net.Conn, error) {
	cfg, err := fleet.GetClientTlsConfig()
	if err != nil {
		cfg = nil
		//return nil, err
	}
	return tls.Dial(network, addr, cfg)
}
