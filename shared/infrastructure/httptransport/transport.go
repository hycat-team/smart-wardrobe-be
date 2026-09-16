package httptransport

import (
	"net/http"
	"time"
)

var defaultTransport *http.Transport

func init() {
	defaultTransport = NewTransport()
}

func NewTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}
}

func DefaultTransport() *http.Transport {
	return defaultTransport
}

func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: NewTransport(),
		Timeout:   timeout,
	}
}
