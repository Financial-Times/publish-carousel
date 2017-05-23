package cluster

import (
	"fmt"
	"net/url"
	"strings"
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
	environments map[string]readEnvironment
}

func newEnvironmentService(readURLs string, credentials string) (*environmentService, error) {
	environments, err := parseEnvironments(readURLs, credentials)
	if err != nil {
		return nil, err
	}
	return &environmentService{environments: environments}, nil
}

func (r *environmentService) GetEnvironments() []readEnvironment {
	var envs []readEnvironment
	for _, env := range r.environments {
		envs = append(envs, env)
	}
	return envs
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
