package cluster

import (
	"context"
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
	readService.startWatcher(context.Background())

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
		if env.readURL == nil {
			log.WithField("name", env.name).Error("Partial information found for environment! Please confirm the etcd value for monitoring read-urls is setup correctly.")
			continue
		}

		log.WithField("gtg", createGTG(env, e.Name())).Info("Calling GTG for external service.")

		req, err := http.NewRequest("GET", createGTG(env, e.Name()), nil)
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
			err := fmt.Errorf("GTG for %v@%v returned a non-200 code: %v", e.Name(), createGTG(env, e.Name()), resp.StatusCode)
			log.WithError(err).Warn("GTG failed for external dependency.")
			errs = append(errs, err)
		}
	}

	return compactErrors(errs)
}

func createGTG(env readEnvironment, name string) string {
	return env.readURL.String() + "/__" + name + "/__gtg"
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
