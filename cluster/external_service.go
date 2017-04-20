package cluster

import (
	"sync"
)

type externalService struct {
	sync.RWMutex
	name         string
	environments []readEnvironment
}

// NewExternalServices returns a new instance of a UpP cluster service
// func NewExternalServices(watcher etcd.Watcher, readUrlsKey string, credentialsKey string) ([]Service, error) {
//
// }

// func (e *externalService) Name() string {
// 	return e.name
// }
//
// func (e *externalService) GTG() error {
// 	e.RLock()
// 	defer e.RUnlock()
//
// 	req, err := http.NewRequest("GET", e.gtgURL.String(), nil)
// 	if err != nil {
// 		return err
// 	}
//
// 	req.SetBasicAuth(e.authUser, e.authPassword)
// 	resp, err := http.DefaultClient.Do(req)
//
// 	if err != nil {
// 		log.WithError(err).WithField("service", e.Name()).Error("Failed to call the GTG endpoint of the service")
// 		return err
// 	}
//
// 	if resp.StatusCode != http.StatusOK {
// 		err := fmt.Errorf("GTG for %v returned a non-200 code: %v", e.Name(), resp.StatusCode)
// 		log.WithError(err).Warn("GTG failed for external dependency.")
// 		return err
// 	}
// 	return nil
// }
