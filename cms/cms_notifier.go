package cms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/native"
	log "github.com/Sirupsen/logrus"
)

// Notifier handles the publishing of the content to the cms-notifier
type Notifier interface {
	Notify(origin string, tid string, content *native.Content, hash string) error
	GTG() error
}

type cmsNotifier struct {
	cluster.Service
	client      *http.Client
	notifierURL string
}

// NewNotifier returns a new cms notifier instance
func NewNotifier(notifierURL string, client *http.Client) (Notifier, error) {
	s, err := cluster.NewService("cms-notifier", notifierURL)
	if err != nil {
		return nil, err
	}
	return &cmsNotifier{s, client, notifierURL}, nil
}

const notifyPath = "/notify"

func (c *cmsNotifier) Notify(origin string, tid string, content *native.Content, hash string) error {
	b := new(bytes.Buffer)

	enc := json.NewEncoder(b)
	err := enc.Encode(content.Body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.notifierURL+notifyPath, b)
	req.Header.Add("Content-Type", content.ContentType)
	req.Header.Add("X-Request-Id", tid)
	req.Header.Add("X-Native-Hash", hash)
	req.Header.Add("X-Origin-System-Id", origin)

	log.WithField("transaction_id", tid).WithField("nativeHash", hash).Info("Calling CMS notifier.")

	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	dump, _ := httputil.DumpResponse(resp, true)
	log.Info(string(dump))

	return fmt.Errorf("A non 2xx error code was received by the CMS Notifier! Status: %v", resp.StatusCode)
}
