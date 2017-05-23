package cluster

import (
	"time"
	"net/http"
)

const requestTimeout = 4500

var client *http.Client

func init() {
	client = &http.Client{Timeout: requestTimeout * time.Millisecond}
}
