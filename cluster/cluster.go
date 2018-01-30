package cluster

import (
	"net/http"
	"time"
)

const requestTimeout = 4500

var client HttpClient

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func init() {
	client = &http.Client{Timeout: requestTimeout * time.Millisecond}
}
