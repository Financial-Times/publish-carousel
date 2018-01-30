package cluster

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/sirupsen/logrus"
)

const healthPath = "/__health"

// Service is a generic service of an UPP cluster that implements a standard FT Good-To-Go endpoint.
type Service interface {
	fmt.Stringer
	ServiceName() string
	Name() string
	Check() error
}

type clusterService struct {
	client            HttpClient
	serviceName       string
	gtgURL            *url.URL
	healthURL         *url.URL
	checkHealthchecks bool
}

// NewService returns a new instance of a UPP cluster service which checks either the /__gtg or the /__health endpoints
func NewService(serviceName string, urlString string, checkHealthchecks bool) (Service, error) {
	gtgURL, err := url.ParseRequestURI(urlString + httphandlers.GTGPath)
	if err != nil {
		return nil, err
	}

	healthURL, err := url.ParseRequestURI(urlString + healthPath)
	if err != nil {
		return nil, err
	}

	return &clusterService{client: client, serviceName: serviceName, gtgURL: gtgURL, healthURL: healthURL, checkHealthchecks: checkHealthchecks}, nil
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

func (s *clusterService) Check() error {
	if s.checkHealthchecks {
		return s.health()
	}
	return s.gtg()
}

func (s *clusterService) health() error {
	resp, err := s.doGet(s.healthURL.String())
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(`Non-200 code while checking "%v"`, s.healthURL.String())
	}

	dec := json.NewDecoder(resp.Body)
	result := make(map[string]interface{})
	err = dec.Decode(&result)

	if err != nil {
		return err
	}

	if !result["ok"].(bool) {
		return fmt.Errorf(`Healthcheck failed @ "%v"`, s.healthURL.String())
	}

	return nil
}

func (s *clusterService) gtg() error {
	resp, err := s.doGet(s.gtgURL.String())
	if err != nil {
		log.WithError(err).WithField("service", s.serviceName).Error("Failed to call the GTG endpoint of the service")
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("GTG for %v returned a non-200 code: %v", s.ServiceName(), resp.StatusCode)
		log.WithError(err).Warn("GTG failed for external dependency.")
		return err
	}

	return nil
}

func (s *clusterService) doGet(serviceUrl string) (*http.Response, error) {
	req, err := http.NewRequest("GET", serviceUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", "UPP Publish Carousel")
	return s.client.Do(req)
}
