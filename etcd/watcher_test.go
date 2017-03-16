package etcd

import (
	"context"
	"testing"

	etcdClient "github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping etcd integration test")
	}

	watcher, err := NewEtcdWatcher([]string{"http://localhost:2379"})
	assert.NoError(t, err)

	go watcher.Watch("/ft/cluster/health/test_key", func(val string) {
		t.Log(val)
		assert.Equal(t, "testing 1 2 3", val)
	})

	cfg := etcdClient.Config{
		Endpoints: []string{"http://localhost:2379"},
	}

	client, err := etcdClient.New(cfg)
	assert.NoError(t, err)

	api := etcdClient.NewKeysAPI(client)
	api.Set(context.Background(), "/ft/cluster/health/test_key", "testing 1 2 3", nil)
}
