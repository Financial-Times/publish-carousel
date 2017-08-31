package file

import (
	"testing"
	"fmt"
	"net/url"

	"github.com/stretchr/testify/assert"
	"github.com/Financial-Times/publish-carousel/cluster"
)

func TestParseEnvironmentsSuccessfully(t *testing.T) {
	testCases := []struct {
		description      string
		readURLs         string
		credentials      string
		expectedEnvsSize int
		expectedEnvSvc   environmentService
	}{
		{
			description:      "Valid readURLs string amd empty credentials string",
			readURLs:         "environment1:http://address1,environment2:http://address2",
			credentials:      "",
			expectedEnvsSize: 2,
			expectedEnvSvc: environmentService{
				environments: map[string]readEnvironment{
					"environment1": {
						name: "environment1",
						readURL: &url.URL{
							Host:   "address1",
							Scheme: "http",
						},
					},
					"environment2": {
						name: "environment2",
						readURL: &url.URL{
							Host:   "address2",
							Scheme: "http",
						},
					},
				},
			},
		},
		{
			description:      "Valid readURLs and credentials string",
			readURLs:         "environment1:https://address1,environment2:http://address2",
			credentials:      "environment1:user:password",
			expectedEnvsSize: 2,
			expectedEnvSvc: environmentService{
				environments: map[string]readEnvironment{
					"environment1": {
						name: "environment1",
						readURL: &url.URL{
							Host:   "address1",
							Scheme: "https",
						},
						credentials: &credentials{
							username: "user",
							password: "password",
						},
					},
					"environment2": {
						name: "environment2",
						readURL: &url.URL{
							Host:   "address2",
							Scheme: "http",
						},
					},
				},
			},
		},
		{
			description:      "readURLs and credentials strings contain only spaces",
			readURLs:         " ",
			credentials:      " ",
			expectedEnvsSize: 0,
			expectedEnvSvc: environmentService{
				environments: map[string]readEnvironment{},
			},
		},
		{
			description:      "readURLs and credentials strings are empty",
			readURLs:         "",
			credentials:      "",
			expectedEnvsSize: 0,
			expectedEnvSvc: environmentService{
				environments: map[string]readEnvironment{},
			},
		},
		{
			description:      "readURLs and credentials strings contain only commas",
			readURLs:         ",,",
			credentials:      ",,",
			expectedEnvsSize: 0,
			expectedEnvSvc: environmentService{
				environments: map[string]readEnvironment{},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.description), func(t *testing.T) {
			envs, err := parseEnvironments(tc.readURLs, tc.credentials)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedEnvsSize, len(envs))
			for expectedEnvName, expectedEnv := range tc.expectedEnvSvc.environments {
				assert.Contains(t, envs, expectedEnvName)
				assert.Equal(t, expectedEnv.name, envs[expectedEnvName].name)
				assert.Equal(t, expectedEnv.credentials, envs[expectedEnvName].credentials)
				assert.Equal(t, expectedEnv.readURL.Host, envs[expectedEnvName].readURL.Host)
				assert.Equal(t, expectedEnv.readURL.Scheme, envs[expectedEnvName].readURL.Scheme)
			}
		})
	}
}

func TestParseEnvironmentsWithFailure(t *testing.T) {
	testCases := []struct {
		readURLs         string
		credentials      string
		description      string
		expectedEnvsSize int
		expectedEnvs     environmentService
	}{
		{
			readURLs:         "environment",
			credentials:      "",
			description:      "readURLs string is missing the environment url",
			expectedEnvsSize: 0,
			expectedEnvs: environmentService{
				environments: map[string]readEnvironment{},
			},
		},
		{
			readURLs:         "",
			credentials:      "environment:user",
			description:      "credentials string is missing the password",
			expectedEnvsSize: 0,
			expectedEnvs: environmentService{
				environments: map[string]readEnvironment{},
			},
		},
		{
			readURLs:         "",
			credentials:      "environment",
			description:      "credentials string is missing user and password",
			expectedEnvsSize: 0,
			expectedEnvs: environmentService{
				environments: map[string]readEnvironment{},
			},
		},
		{
			readURLs:         "environment1:http://[::1]:namedport,environment2:http://localhost",
			credentials:      "",
			description:      "readURLs string contains an invalid url",
			expectedEnvsSize: 1,
			expectedEnvs: environmentService{
				environments: map[string]readEnvironment{
					"environment2": {
						name: "environment2",
						readURL: &url.URL{
							Host:   "localhost",
							Scheme: "http",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.description), func(t *testing.T) {
			envs, err := parseEnvironments(tc.readURLs, tc.credentials)
			assert.Error(t, err)
			assert.Equal(t, tc.expectedEnvsSize, len(envs))
			if tc.expectedEnvsSize != 0 {
				for expectedEnvName, expectedEnv := range tc.expectedEnvs.environments {
					assert.Contains(t, envs, expectedEnvName)
					env := envs[expectedEnvName]
					assertEnvironment(t, &expectedEnv, &env)
				}
			}
		})
	}
}

func TestGetEnvironments(t *testing.T) {
	expectedReadEnvs := []readEnvironment{
		{
			name: "environment1",
			readURL: &url.URL{
				Host:   "address1",
				Scheme: "http",
			},
			credentials: &credentials{
				username: "user",
				password: "password",
			}},
		{
			name: "environment2",
			readURL: &url.URL{
				Host:   "address2",
				Scheme: "http",
			},
			credentials: &credentials{
				username: "user",
				password: "password",
			}},
	}

	watcher := new(cluster.MockWatcher)
	envSrv, _ := newEnvironmentService(watcher, "file1", "file2")
	//envSrv := environmentService{watcher, map[string]readEnvironment{
	//	"environment1": expectedReadEnvs[0],
	//	"environment2": expectedReadEnvs[1],
	//},
	//}

	envs := envSrv.GetEnvironments()

	assert.Equal(t, 2, len(envs))
	assert.Contains(t, envs, expectedReadEnvs[0])
	assert.Contains(t, envs, expectedReadEnvs[1])
}

func TestNewEnvironmentService(t *testing.T) {
	testCases := []struct {
		description    string
		expectError    bool
		readURLs       string
		credentials    string
		expectedEnvSvc environmentService
	}{
		{
			description: "Valid environment and credential strings",
			expectError: false,
			readURLs:    "environment1:https://address1,environment2:http://address2",
			credentials: "environment1:user1:password1,environment2:user2:password2",
			expectedEnvSvc: environmentService{
				environments: map[string]readEnvironment{
					"environment1": {
						name: "environment1",
						readURL: &url.URL{
							Host:   "address1",
							Scheme: "https",
						},
						credentials: &credentials{
							username: "user1",
							password: "password1",
						},
					},
					"environment2": {
						name: "environment2",
						readURL: &url.URL{
							Host:   "address2",
							Scheme: "http",
						},
						credentials: &credentials{
							username: "user2",
							password: "password2",
						},
					},
				},
			},
		},
		{
			description: "First environment is missing colons",
			expectError: true,
			readURLs:    "environment1http//address1,environment2:http://address2",
			credentials: "",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.description), func(t *testing.T) {
			watcher := new(cluster.MockWatcher)
			envSvc, err := newEnvironmentService(watcher, tc.readURLs, tc.credentials)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.expectedEnvSvc.environments), len(envSvc.environments))
				for key, value := range tc.expectedEnvSvc.environments {
					assert.Contains(t, envSvc.environments, key)
					env := envSvc.environments[key]
					assertEnvironment(t, &value, &env)
				}
			}
		})
	}
}

func assertEnvironment(t *testing.T, expectedEnv *readEnvironment, env *readEnvironment) {
	assert.Equal(t, expectedEnv.name, env.name)
	assert.Equal(t, expectedEnv.credentials, env.credentials)
	assert.Equal(t, expectedEnv.readURL.Host, env.readURL.Host)
	assert.Equal(t, expectedEnv.readURL.Scheme, env.readURL.Scheme)
}
