package file

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/Financial-Times/publish-carousel/file"
	log "github.com/Sirupsen/logrus"
)

type readEnvironment struct {
	name        string
	readURL     *url.URL
	credentials *credentials
}

type credentials struct {
	username string
	password string
}

// NewReadCluster parse read cluster urls and credentials
type environmentService struct {
	sync.RWMutex
	watcher      file.Watcher
	environments map[string]readEnvironment
}

func newEnvironmentService(watcher file.Watcher, readEnvironmentsFile string, credentialsFile string) (*environmentService, error) {
	readURLs, err := watcher.Read(readEnvironmentsFile)
	if err != nil {
		log.WithField("file", readEnvironmentsFile).Error("Cannot read environments from file: ", err)
		return nil, err
	}
	credentialsString := ""
	if credentialsFile != "" {
		credentialsString, err = watcher.Read(credentialsFile)
		if err != nil {
			log.WithField("file", credentialsFile).Error("Cannot read credentials from file: ", err)
			return nil, err
		}
	}
	environments, err := parseEnvironments(readURLs, credentialsString)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	return &environmentService{environments: environments, watcher: watcher}, nil
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

func (r *environmentService) startWatcher(ctx context.Context, readEnvironmentsFile string, credentialsFile string) {
	//we have to watch both environments and credentials as we don't know which ones will change
	go r.watcher.Watch(ctx, readEnvironmentsFile, func(readEnvironments string) {
		r.Lock()
		defer r.Unlock()

		credentials, err := r.watcher.Read(credentialsFile)
		if err != nil {
			log.WithError(err).Error("Environments were updated but failed to read credentials file!")
			return
		}
		update, err := parseEnvironments(readEnvironments, credentials)
		if err != nil {
			log.WithError(err).Error("One or more read-urls failed validation!")
		}
		r.environments = update
	})

	if credentialsFile == "" {
		return
	}

	go r.watcher.Watch(ctx, credentialsFile, func(credentials string) {
		r.Lock()
		defer r.Unlock()

		readEnvironments, err := r.watcher.Read(readEnvironmentsFile)
		if err != nil {
			log.WithError(err).Error("CredentialsFile were updated but failed to read environments file!")
			return
		}
		update, err := parseEnvironments(readEnvironments, credentials)
		if err != nil {
			log.WithError(err).Error("One or more read-urls failed validation!")
		}
		r.environments = update
	})
}

func parseEnvironments(readURLs string, credentialsString string) (map[string]readEnvironment, error) {
	envMap := make(map[string]readEnvironment)

	credentialsMap := make(map[string]*credentials)
	if strings.TrimSpace(credentialsString) != "" {
		for _, creds := range strings.Split(credentialsString, ",") {
			if strings.TrimSpace(creds) == "" {
				continue
			}

			envAndCredentials := strings.SplitN(creds, ":", 3)
			if len(envAndCredentials) != 3 {
				return envMap, fmt.Errorf(`The provided creds string "%v" is invalid - should be in the format environment1:user:pass,environment2:user:pass`, credentialsString)
			}

			credentialsMap[envAndCredentials[0]] = &credentials{envAndCredentials[1], envAndCredentials[2]}
		}
	}

	if strings.TrimSpace(readURLs) == "" {
		return envMap, nil
	}

	environments := strings.Split(readURLs, ",")

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

		envMap[envAndURL[0]] = readEnvironment{name: envAndURL[0], readURL: uri, credentials: credentialsMap[envAndURL[0]]}
	}

	return envMap, compactErrors("One or more read-urls failed validation", errs...)
}
