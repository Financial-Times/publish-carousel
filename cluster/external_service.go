package cluster

import (
	"errors"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

type externalService struct {
	name               string
	serviceName        string
	environmentService *environmentService
}

// NewExternalService returns a new instance of a UPP cluster service which is in an external cluster (i.e. delivery)
func NewExternalService(name string, serviceName string, readURLs string, credentials string) (Service, error) {
	environmentService, err := newEnvironmentService(readURLs, credentials)
	return &externalService{name: name, serviceName: serviceName, environmentService: environmentService}, err
}

func (e *externalService) Name() string {
	return e.name
}

func (e *externalService) ServiceName() string {
	return e.serviceName
}

func (e *externalService) Check() error {
	envs := e.environmentService.GetEnvironments()

	errs := make([]error, 0)
	for _, env := range envs {
		gtg := gtgURLFor(env, e.ServiceName())
		log.WithField("gtg", gtg).Info("Calling GTG for external service.")

		req, err := http.NewRequest("GET", gtg, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if env.credentials != nil {
			req.SetBasicAuth(env.credentials.username, env.credentials.password)
		}

		resp, err := client.Do(req)

		if err != nil {
			log.WithError(err).WithField("service", e.ServiceName()).Error("Failed to call the GTG endpoint of the service")
			errs = append(errs, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("GTG for %v@%v returned a non-200 code: %v", e.ServiceName(), gtg, resp.StatusCode)
			log.WithError(err).Warn("GTG failed for external dependency.")
			errs = append(errs, err)
		}

		log.WithField("gtg", gtg).WithField("status", resp.StatusCode).Info("GTG succeeded for external service.")
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
