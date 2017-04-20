package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/etcd"
	etcdClient "github.com/coreos/etcd/client"

	"github.com/stretchr/testify/assert"
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

func TestSetupReadCluster(t *testing.T) {
	// if testing.Short() {
	// t.Skip("Skipping etcd integration test")
	// }

	watcher, err := etcd.NewEtcdWatcher([]string{"http://localhost:2379"})
	assert.NoError(t, err)

	ctx := context.Background()

	assert.NoError(t, err)

	api.Set(ctx, "/test1/ft/config/monitoring/read-urls", "dynpub:http://localhost:8080,semantic:http://localhost:9090", nil)
	api.Set(ctx, "/test1/ft/_credentials/publish-read/read-credentials", "dynpub:dynpub:my-garbage-fake-password,semantic:user:pass", nil)

	readCluster, err := newReadCluster(watcher, "/test1/ft/config/monitoring/read-urls", "/test1/ft/_credentials/publish-read/read-credentials")
	assert.NoError(t, err)

	envs := readCluster.GetReadEnvironments()
	assert.Len(t, envs, 2)

	dynpub := envs[0]
	semantic := envs[1]

	if envs[0].name != "dynpub" {
		dynpub = envs[1]
		semantic = envs[0]
	}

	assert.Equal(t, "dynpub", dynpub.name)
	assert.Equal(t, "http://localhost:8080", dynpub.readURL.String())
	assert.Equal(t, "dynpub", dynpub.authUser)
	assert.Equal(t, "my-garbage-fake-password", dynpub.authPassword)

	assert.Equal(t, "semantic", semantic.name)
	assert.Equal(t, "http://localhost:9090", semantic.readURL.String())
	assert.Equal(t, "user", semantic.authUser)
	assert.Equal(t, "pass", semantic.authPassword)
}

func TestWatchReadClusterChanges(t *testing.T) {
	// if testing.Short() {
	// t.Skip("Skipping etcd integration test")
	// }

	watcher, err := etcd.NewEtcdWatcher([]string{"http://localhost:2379"})
	assert.NoError(t, err)

	readCluster, err := newReadCluster(watcher, "/test2/ft/config/monitoring/read-urls", "/test2/ft/_credentials/publish-read/read-credentials")
	assert.NoError(t, err)

	api.Set(context.Background(), "/test2/ft/config/monitoring/read-urls", "dynpub:http://localhost:8080", nil)
	api.Set(context.Background(), "/test2/ft/_credentials/publish-read/read-credentials", "dynpub:dynpub:test2,semantic:user:pass", nil)

	timeout, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer timeoutCancel()
	readCluster.startWatcher(timeout)

	envs := readCluster.GetReadEnvironments()
	assert.Len(t, envs, 1)

	go func() {
		time.Sleep(1 * time.Second)
		api.Set(context.Background(), "/test2/ft/config/monitoring/read-urls", "dynpub:http://localhost:8088,semantic:http://localhost:9099", nil)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	watcher.Watch(ctx, "/test2/ft/config/monitoring/read-urls", func(val string) {
		assert.Equal(t, "dynpub:http://localhost:8088,semantic:http://localhost:9099", val)
		cancel()
	})

	envs = readCluster.GetReadEnvironments()
	assert.Len(t, envs, 2)

	dynpub := envs[0]
	semantic := envs[1]

	if envs[0].name != "dynpub" {
		dynpub = envs[1]
		semantic = envs[0]
	}

	assert.Equal(t, "dynpub", dynpub.name)
	assert.Equal(t, "http://localhost:8088", dynpub.readURL.String())
	assert.Equal(t, "dynpub", dynpub.authUser)
	assert.Equal(t, "test2", dynpub.authPassword)

	assert.Equal(t, "semantic", semantic.name)
	assert.Equal(t, "http://localhost:9099", semantic.readURL.String())
	assert.Equal(t, "user", semantic.authUser)
	assert.Equal(t, "pass", semantic.authPassword)
}
