package etcd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/etcd"
	log "github.com/sirupsen/logrus"
)

type externalService struct {
	sync.RWMutex
	name               string
	serviceName        string
	environmentService *environmentService
}

// NewExternalService returns a new instance of a UPP cluster service which is in an external cluster (i.e. delivery)
func NewExternalService(name string, serviceName string, watcher etcd.Watcher, readURLsKey string) (cluster.Service, error) {
	environmentService, err := newEnvironmentService(watcher, readURLsKey)
	environmentService.startWatcher(context.Background())

	return &externalService{name: name, serviceName: serviceName, environmentService: environmentService}, err
}

func (e *externalService) Name() string {
	return e.name
}

func (e *externalService) ServiceName() string {
	return e.serviceName
}

func (e *externalService) Check() error {
	e.RLock()
	defer e.RUnlock()

	envs := e.environmentService.GetEnvironments()

	errs := make([]error, 0)
	for _, env := range envs {
		gtg := gtgURLFor(env, e.ServiceName())
		log.WithField("gtg", gtg).Debug("Calling GTG for external service.")

		req, err := http.NewRequest("GET", gtg, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		req.Header.Add("User-Agent", "UPP Publish Carousel")
		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			log.WithError(err).WithField("service", e.ServiceName()).Warn("Failed to call the GTG endpoint of the service")
			errs = append(errs, err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("GTG for %v@%v returned a non-200 code: %v", e.ServiceName(), gtg, resp.StatusCode)
			log.WithError(err).Warn("GTG failed for external dependency.")
			errs = append(errs, err)
			continue
		}

		log.WithField("gtg", gtg).WithField("status", resp.StatusCode).Debug("GTG succeeded for external service.")
	}

	return compactErrors("Failure occurred while checking GTG for external service.", errs...)
}

func (e *externalService) String() string {
	envs := e.environmentService.GetEnvironments()

	desc := e.name + " -"
	for _, env := range envs {
		desc += " " + env.name + ": " + env.readURL.String() + ","
	}
	return desc
}

func gtgURLFor(env readEnvironment, serviceName string) string {
	return env.readURL.String() + "/__" + serviceName + "/__gtg"
}

func compactErrors(msg string, errs ...error) error {
	if len(errs) == 0 {
		return nil
	}

	for _, err := range errs {
		msg += "\n" + err.Error()
	}

	return errors.New(msg)
}
