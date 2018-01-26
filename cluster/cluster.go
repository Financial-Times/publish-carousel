package cluster

import (
	"net/http"
	"time"
)

const requestTimeout = 4500

var client httpClient

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func init() {
	client = &http.Client{Timeout: requestTimeout * time.Millisecond}
}
