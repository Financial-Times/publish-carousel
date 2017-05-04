package cluster

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
)

// Service is a generic service of an UPP cluster that implements a standard FT Good-To-Go endpoint.
type Service interface {
	fmt.Stringer
	ServiceName() string
	Name() string
	GTG() error
}

type clusterService struct {
	serviceName string
	gtgURL      *url.URL
}

// NewService returns a new instance of a UPP cluster service
func NewService(serviceName string, urlString string) (Service, error) {
	gtgURL, err := url.ParseRequestURI(urlString + httphandlers.GTGPath)
	if err != nil {
		return nil, err
	}
	return &clusterService{serviceName, gtgURL}, nil
}

func (s *clusterService) Name() string {
	return s.serviceName
}

func (s *clusterService) ServiceName() string {
	return s.serviceName
}

func (s *clusterService) String() string {
	return s.gtgURL.String()
}

func (s *clusterService) GTG() error {
	resp, err := http.Get(s.gtgURL.String())
	if err != nil {
		log.WithError(err).WithField("service", s.serviceName).Error("Failed to call the GTG endpoint of the service")
		return err
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("GTG for %v returned a non-200 code: %v", s.ServiceName(), resp.StatusCode)
		log.WithError(err).Warn("GTG failed for external dependency.")
		return err
	}
	return nil
}
