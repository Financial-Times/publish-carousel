package cluster

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"time"
)

var client *http.Client

const requestTimeout = 5

func init() {
	client = &http.Client{Timeout: requestTimeout * time.Second}
}

// Service is a generic service of an UP cluster that implements a standard FT Good-To-Go endpoint.
type Service interface {
	GTG() error
	Name() string
}

type clusterService struct {
	name   string
	gtgURL *url.URL
}

// NewService returns a new instance of a UpP cluster service
func NewService(serviceName string, urlString string) (Service, error) {
	gtgURL, err := url.ParseRequestURI(urlString + httphandlers.GTGPath)
	if err != nil {
		return nil, err
	}
	return &clusterService{serviceName, gtgURL}, nil
}

func (s *clusterService) Name() string {
	return s.name
}

func (s *clusterService) GTG() error {
	resp, err := client.Get(s.gtgURL.String())
	if err != nil {
		log.WithError(err).WithField("service", s.name).Error("Failed to call the GTG endpoint of the service")
		return err
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("GTG for %v returned a non-200 code: %v", s.Name(), resp.StatusCode)
		log.WithError(err).Warn("GTG failed for external dependency.")
		return err
	}
	return nil
}
