package cluster

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/Financial-Times/publish-carousel/etcd"
	log "github.com/Sirupsen/logrus"
)

type readEnvironment struct {
	name         string
	readURL      *url.URL
	authUser     string
	authPassword string
}

type readService struct {
	sync.RWMutex
	watcher        etcd.Watcher
	readURLsKey    string
	credentialsKey string
	environments   map[string]readEnvironment
}

// NewReadCluster parse read cluster urls and credentials
func newReadService(watcher etcd.Watcher, readURLsKey string, credentialsKey string) (*readService, error) {
	urls, err := watcher.Read(readURLsKey)
	if err != nil {
		return nil, err
	}

	environmentMap, err := parseEnvironmentsToSet(urls)
	if err != nil {
		return nil, err
	}

	credentials, err := watcher.Read(credentialsKey)
	if err != nil {
		return nil, err
	}

	all := make(map[string]readEnvironment)
	for envName := range environmentMap {
		readEnv := &readEnvironment{name: envName}
		err = readEnv.updateReadURLs(urls)
		if err != nil {
			log.WithField("name", envName).WithField("etcdKey", readURLsKey).WithField("readUrls", urls).Warn("Failed to parse read urls from etcd key.")
			continue
		}

		err = readEnv.updateCredentials(credentials)
		if err != nil {
			log.WithField("name", envName).WithField("etcdKey", credentialsKey).Warn("Failed to parse read credentials from etcd key.")
			return nil, err // fail fast for credentials issue
		}

		all[envName] = *readEnv
	}

	return &readService{environments: all, watcher: watcher, readURLsKey: readURLsKey, credentialsKey: credentialsKey}, nil
}

func (r *readService) GetReadEnvironments() []readEnvironment {
	r.RLock()
	defer r.RUnlock()

	var envs []readEnvironment
	for _, env := range r.environments {
		// log.Info(env.authUser + env.authPassword + env.readURL.String() + env.name)
		envs = append(envs, env)
	}
	return envs
}

func (r *readService) startWatcher(ctx context.Context) {
	go r.watcher.Watch(ctx, r.readURLsKey, func(readURLs string) {
		r.Lock()
		defer r.Unlock()

		updated, err := parseEnvironmentsToSet(readURLs)
		if err != nil {
			log.WithField("etcdKey", r.readURLsKey).WithField("readUrls", readURLs).Warn("Failed to parse updated etcd key for the read environment")
			return
		}

		for name := range updated {
			env, ok := r.environments[name]
			if !ok {
				env = readEnvironment{name: name}
			}

			e := &env
			e.updateReadURLs(readURLs)
			r.environments[name] = *e
		}

		for _, env := range r.environments {
			if _, ok := updated[env.name]; !ok {
				delete(r.environments, env.name)
			}
		}
	})

	go r.watcher.Watch(ctx, r.credentialsKey, func(credentials string) {
		r.Lock()
		defer r.Unlock()

		updated, err := parseEnvironmentsToSet(credentials)
		if err != nil {
			log.WithField("etcdKey", r.credentialsKey).Warn("Failed to parse updated etcd key for the read credentials")
			return
		}

		for name := range updated {
			env, ok := r.environments[name]
			if !ok {
				env = readEnvironment{name: name}
			}

			e := &env
			e.updateCredentials(credentials)
			r.environments[name] = *e
		}
	})
}

func parseEnvironmentsToSet(readURLs string) (map[string]struct{}, error) {
	envMap := make(map[string]struct{})
	if strings.TrimSpace(readURLs) == "" {
		return envMap, nil
	}

	environments := strings.Split(readURLs, ",")

	for _, environment := range environments {
		env := strings.SplitN(environment, ":", 2)
		if len(env) != 2 {
			return envMap, fmt.Errorf(`config for read environment "%v" is incorrect - should be in the format environment:read-url`, environment)
		}
		envMap[env[0]] = struct{}{}
	}

	return envMap, nil
}

func (e *readEnvironment) updateReadURLs(readUrlsVal string) error {
	environments := strings.Split(readUrlsVal, ",")

	for _, environment := range environments {
		if strings.HasPrefix(environment, e.name+":") {
			envAndURL := strings.SplitN(environment, ":", 2)
			if len(envAndURL) != 2 {
				log.WithField("name", e.name).WithField("readUrl", readUrlsVal).Warn("Incorrect config found for external service!")
				return errors.New("failed to parse read url")
			}

			uri, err := url.Parse(envAndURL[1])
			if err != nil {
				return err
			}

			e.readURL = uri
			break
		}
	}

	return nil
}

func (e *readEnvironment) updateCredentials(credentialsVal string) error {
	credentials := strings.Split(credentialsVal, ",")

	for _, credential := range credentials {
		if strings.HasPrefix(credential, e.name+":") {
			creds := strings.Split(credential, ":")
			if len(creds) != 3 {
				log.WithField("name", e.name).Warn("Incorrect credentials found for external service!")
				return errors.New("Failed to parse credentials")
			}

			e.authUser = creds[1]
			e.authPassword = creds[2]
			return nil
		}
	}

	return fmt.Errorf(`No credentials for environment "%v"`, e.name)
}
