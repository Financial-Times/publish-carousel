package cluster

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/Financial-Times/publish-carousel/etcd"
	log "github.com/Sirupsen/logrus"
)

type readEnvironment struct {
	name    string
	readURL *url.URL
}

type environmentService struct {
	sync.RWMutex
	watcher      etcd.Watcher
	readURLsKey  string
	environments map[string]readEnvironment
}

// NewReadCluster parse read cluster urls and credentials
func newEnvironmentService(watcher etcd.Watcher, readURLsKey string) (*environmentService, error) {
	urls, err := watcher.Read(readURLsKey)
	if err != nil {
		return nil, err
	}

	environments, err := parseEnvironments(urls)
	if err != nil {
		return nil, err
	}

	return &environmentService{environments: environments, watcher: watcher, readURLsKey: readURLsKey}, nil
}

func (r *environmentService) GetEnvironments() []readEnvironment {
	r.RLock()
	defer r.RUnlock()

	var envs []readEnvironment
	for _, env := range r.environments {
		envs = append(envs, env)
	}
	return envs
}

func (r *environmentService) startWatcher(ctx context.Context) {
	go r.watcher.Watch(ctx, r.readURLsKey, func(readURLs string) {
		r.Lock()
		defer r.Unlock()

		update, err := parseEnvironments(readURLs)
		if err != nil {
			log.WithError(err).Error("One or more read-urls failed validation!")
		}

		r.environments = update
	})
}

func parseEnvironments(value string) (map[string]readEnvironment, error) {
	envMap := make(map[string]readEnvironment)
	if strings.TrimSpace(value) == "" {
		return envMap, nil
	}

	environments := strings.Split(value, ",")

	errs := make([]error, 0)
	for _, environment := range environments {
		if strings.TrimSpace(environment) == "" {
			continue
		}

		envAndURL := strings.SplitN(environment, ":", 2)
		if len(envAndURL) != 2 {
			return envMap, fmt.Errorf(`config for environment "%v" is incorrect - should be in the format environment1:*,environment2:*`, environment)
		}

		uri, err := url.Parse(envAndURL[1])
		if err != nil {
			errs = append(errs, fmt.Errorf("%v - %v", envAndURL[0], err.Error()))
			continue
		}

		envMap[envAndURL[0]] = readEnvironment{name: envAndURL[0], readURL: uri}
	}

	return envMap, compactErrors("One or more read-urls failed validation", errs...)
}
