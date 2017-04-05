package cluster

import (
	"fmt"
	"net/http"
	"net/url"

	log "github.com/Sirupsen/logrus"
)

const gtgtPath = "/__gtg"

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
	gtgURL, err := url.ParseRequestURI(urlString + gtgtPath)
	if err != nil {
		return nil, err
	}
	return &clusterService{serviceName, gtgURL}, nil
}

func (s *clusterService) Name() string {
	return s.name
}

func (s *clusterService) GTG() error {
	resp, err := http.Get(s.gtgURL.String())
	if err != nil {
		log.WithError(err).WithField("service", s.name).Error("Error in calling the GTG enpoint of the service")
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gtg for %v returned a non-200 code: %v", s.Name(), resp.StatusCode)
	}
	return nil
}
