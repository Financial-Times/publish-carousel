package cluster

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/Financial-Times/publish-carousel/etcd"
	log "github.com/Sirupsen/logrus"
)

type externalService struct {
	sync.RWMutex
	name        string
	readService *readService
}

// NewExternalService returns a new instance of a UpP cluster service
func NewExternalService(name string, watcher etcd.Watcher, readURLsKey string, credentialsKey string) (Service, error) {
	readService, err := newReadService(watcher, readURLsKey, credentialsKey)
	return &externalService{name: name, readService: readService}, err
}

func (e *externalService) Name() string {
	return e.name
}

func (e *externalService) GTG() error {
	e.RLock()
	defer e.RUnlock()

	envs := e.readService.GetReadEnvironments()

	var errs []error
	for _, env := range envs {
		req, err := http.NewRequest("GET", env.readURL.String()+"/__"+e.name+"/__gtg", nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		req.SetBasicAuth(env.authUser, env.authPassword)
		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			log.WithError(err).WithField("service", e.Name()).Error("Failed to call the GTG endpoint of the service")
			errs = append(errs, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("GTG for %v returned a non-200 code: %v", e.Name(), resp.StatusCode)
			log.WithError(err).Warn("GTG failed for external dependency.")
			errs = append(errs, err)
		}
	}

	return compactErrors(errs)
}

func compactErrors(errs []error) error {
	if errs == nil || len(errs) == 0 {
		return nil
	}

	msg := "Failure occurred while checking GTG for external service.\n"
	for _, err := range errs {
		msg = msg + err.Error() + "\n"
	}

	return errors.New(msg)
}
