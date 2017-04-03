package cluster

import (
	"net/http"
	"net/url"

	log "github.com/Sirupsen/logrus"
)

// Service is a generic service of an UP cluster that implements a standard FT Good-To-Go endpoint.
type Service interface {
	GTG() bool
	Name() string
}

type clusterService struct {
	name   string
	gtgURL *url.URL
}

// NewService returns a new instance of a UpP cluster service
func NewService(serviceName string, gtgURLString string) (Service, error) {
	gtgURL, err := url.ParseRequestURI(gtgURLString)
	if err != nil {
		return nil, err
	}
	return &clusterService{serviceName, gtgURL}, nil
}

func (s *clusterService) Name() string {
	return s.name
}

func (s *clusterService) GTG() bool {
	resp, err := http.Get(s.gtgURL.RequestURI())
	if err != nil {
		log.WithError(err).WithField("service", s.name).Error("Error in calling the GTG enpoint of the service")
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}
