package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/etcd"
	etcdClient "github.com/coreos/etcd/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var client etcdClient.Client
var api etcdClient.KeysAPI

func init() {
	cfg := etcdClient.Config{
		Endpoints: []string{"http://localhost:2379"},
	}

	var err error
	client, err = etcdClient.New(cfg)
	if err != nil {
		panic(err)
	}

	api = etcdClient.NewKeysAPI(client)
}

func setupTests(t *testing.T, readURLs string) etcd.Watcher {
	watcher, err := etcd.NewEtcdWatcher([]string{"http://localhost:2379"})
	assert.NoError(t, err)

	ctx := context.Background()

	assert.NoError(t, err)

	api.Set(ctx, readURLs, "environment:http://localhost:8080", nil)
	return watcher
}

func assertEnvironment(t *testing.T, env readEnvironment, name string, url string) {
	assert.Equal(t, name, env.name)
	assert.Equal(t, url, env.readURL.String())
}

func etcdKeys(testID string) string {
	return "/" + testID + "/ft/config/monitoring/read-urls"
}

func TestSetupReadCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping etcd integration test")
	}

	readURLsKey := etcdKeys("test1")
	watcher := setupTests(t, readURLsKey)

	readService, err := newEnvironmentService(watcher, readURLsKey)
	assert.NoError(t, err)
	require.NotNil(t, readService)

	envs := readService.GetEnvironments()
	require.Len(t, envs, 1)

	assertEnvironment(t, envs[0], "environment", "http://localhost:8080")
}

func TestWatchingEtcdKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping etcd integration test")
	}

	readURLsKey := etcdKeys("test2")
	watcher := setupTests(t, readURLsKey)

	readService, err := newEnvironmentService(watcher, readURLsKey)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	readService.startWatcher(ctx)

	envs := readService.GetEnvironments()
	require.Len(t, envs, 1)

	assertEnvironment(t, envs[0], "environment", "http://localhost:8080")

	watchAddingNewEnvChangingDetails(t, watcher, readService, readURLsKey)
	watchRemovingOriginalAddingNew(t, watcher, readService, readURLsKey)
	watchRemovingNewEnvChangingDetails(t, watcher, readService, readURLsKey)
	watchInvalidReadURLValue(t, watcher, readService, readURLsKey)
}

func watchAddingNewEnvChangingDetails(t *testing.T, watcher etcd.Watcher, readService *environmentService, readURLsKey string) {
	go func() { // Validate adding a new environment and changing the original details
		time.Sleep(1 * time.Second)
		api.Set(context.TODO(), readURLsKey, "environment:http://host-changed:8080,environment2:http://host-added:8080", nil)
	}()

	ctx2, cancel2 := context.WithCancel(context.TODO())
	watcher.Watch(ctx2, readURLsKey, func(val string) {
		assert.Equal(t, "environment:http://host-changed:8080,environment2:http://host-added:8080", val)
		cancel2()
	})

	time.Sleep(100 * time.Millisecond)

	envs := readService.GetEnvironments()
	require.Len(t, envs, 2)

	environment := envs[0]
	environment2 := envs[1]

	if envs[0].name != "environment" {
		environment = envs[1]
		environment2 = envs[0]
	}

	assertEnvironment(t, environment, "environment", "http://host-changed:8080")
	assertEnvironment(t, environment2, "environment2", "http://host-added:8080")
}

func watchRemovingNewEnvChangingDetails(t *testing.T, watcher etcd.Watcher, readService *environmentService, readURLsKey string) {
	go func() { // Validate removing the new environment and changing the original environment again
		time.Sleep(1 * time.Second)
		api.Set(context.TODO(), readURLsKey, "environment:http://host-changed-back:8080", nil)
	}()

	ctx, cancel := context.WithCancel(context.TODO())
	watcher.Watch(ctx, readURLsKey, func(val string) {
		assert.Equal(t, "environment:http://host-changed-back:8080", val)
		cancel()
	})

	time.Sleep(100 * time.Millisecond)

	envs := readService.GetEnvironments()
	require.Len(t, envs, 1)
	assertEnvironment(t, envs[0], "environment", "http://host-changed-back:8080")
}

func watchRemovingOriginalAddingNew(t *testing.T, watcher etcd.Watcher, readService *environmentService, readURLsKey string) {
	go func() { // Validate removing the original environment and adding a new one. (reverse order of keys too)
		time.Sleep(1 * time.Second)
		api.Set(context.TODO(), readURLsKey, ",environment2:http://another-host-added:8080", nil)
	}()

	ctx3, cancel3 := context.WithCancel(context.TODO())
	watcher.Watch(ctx3, readURLsKey, func(val string) {
		assert.Equal(t, ",environment2:http://another-host-added:8080", val)
		cancel3()
	})

	time.Sleep(100 * time.Millisecond)

	envs := readService.GetEnvironments()
	require.Len(t, envs, 1)

	environment := envs[0]
	assertEnvironment(t, environment, "environment2", "http://another-host-added:8080")
}

func watchInvalidReadURLValue(t *testing.T, watcher etcd.Watcher, readService *environmentService, readURLsKey string) {
	go func() { // Invalid values
		time.Sleep(1 * time.Second)
		api.Set(context.TODO(), readURLsKey, "environment2::#", nil)
	}()

	ctx4, cancel4 := context.WithCancel(context.TODO())
	watcher.Watch(ctx4, readURLsKey, func(val string) {
		assert.Equal(t, "environment2::#", val)
		cancel4()
	})

	time.Sleep(100 * time.Millisecond)

	envs := readService.GetEnvironments()
	require.Len(t, envs, 0)
}

func TestSetupReadClusterFailsReadURLs(t *testing.T) {
	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("this shouldn't work", nil)

	readService, err := newEnvironmentService(watcher, "read-key")

	assert.Error(t, err)
	assert.Nil(t, readService)
	watcher.AssertExpectations(t)
}

func TestSetupReadClusterSucceedsWithEmptyKeys(t *testing.T) {
	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("", nil)

	readService, err := newEnvironmentService(watcher, "read-key")
	assert.NoError(t, err)

	envs := readService.GetEnvironments()
	assert.Len(t, envs, 0)
	watcher.AssertExpectations(t)
}

func TestReadKeyFails(t *testing.T) {
	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("", errors.New("failed"))

	readService, err := newEnvironmentService(watcher, "read-key")
	assert.Error(t, err)
	assert.Nil(t, readService)
	watcher.AssertExpectations(t)
}

func TestReadURLFails(t *testing.T) {
	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment::#,environment2::#", nil)

	readService, err := newEnvironmentService(watcher, "read-key")
	assert.Error(t, err)
	t.Log(err)
	assert.Nil(t, readService)
	watcher.AssertExpectations(t)
}

func TestReadURLsTrailingComma(t *testing.T) {
	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:localhost,", nil)

	readService, err := newEnvironmentService(watcher, "read-key")
	assert.NoError(t, err)

	envs := readService.GetEnvironments()
	assert.Len(t, envs, 1)
	watcher.AssertExpectations(t)
}
