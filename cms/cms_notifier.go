package cms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/native"
	log "github.com/sirupsen/logrus"
)

// Notifier handles the publishing of the content to the cms-notifier
type Notifier interface {
	Notify(origin string, tid string, content *native.Content, hash string) error
	Check() error
}

type cmsNotifier struct {
	cluster.Service
	client      cluster.HttpClient
	notifierURL string
}

// NewNotifier returns a new cms notifier instance
func NewNotifier(notifierURL string, client cluster.HttpClient) (Notifier, error) {
	s, err := cluster.NewService("cms-notifier", notifierURL, false)
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
	req.Header.Add("User-Agent", "UPP Publish Carousel")
	req.Header.Add("Content-Type", content.ContentType)
	req.Header.Add("X-Request-Id", tid)
	req.Header.Add("X-Native-Hash", hash)
	if content.OriginSystemID != "" {
		origin = content.OriginSystemID
	}
	req.Header.Add("X-Origin-System-Id", origin)
	log.WithField("transaction_id", tid).WithField("nativeHash", hash).Info(fmt.Sprintf("Calling CMS notifier with contentType=%s, Origin=%s", content.ContentType, origin))

	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	dump, _ := httputil.DumpResponse(resp, true)
	log.Info(string(dump))

	return fmt.Errorf("A non 2xx error code was received by the CMS Notifier! Status: %v", resp.StatusCode)
}
