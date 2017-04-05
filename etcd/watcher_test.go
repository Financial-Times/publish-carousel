package etcd

import (
	"context"
	"testing"
	"time"

	etcdClient "github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping etcd integration test")
	}

	watcher, err := NewEtcdWatcher([]string{"http://localhost:2379"})
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)

	cfg := etcdClient.Config{
		Endpoints: []string{"http://localhost:2379"},
	}

	client, err := etcdClient.New(cfg)
	assert.NoError(t, err)

	api := etcdClient.NewKeysAPI(client)
	go func() {
		time.Sleep(3 * time.Second)
		api.Set(context.Background(), "/ft/cluster/health/test_key", "testing 1 2 3", nil)
	}()

	watcher.Watch(ctx, "/ft/cluster/health/test_key", func(val string) {
		assert.Equal(t, "testing 1 2 3", val)
		cancel()
	})
}

func TestWatchCallbackPanics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping etcd integration test")
	}

	watcher, err := NewEtcdWatcher([]string{"http://localhost:2379"})
	assert.NoError(t, err)

	cfg := etcdClient.Config{
		Endpoints: []string{"http://localhost:2379"},
	}

	client, err := etcdClient.New(cfg)
	assert.NoError(t, err)

	api := etcdClient.NewKeysAPI(client)
	go func() {
		time.Sleep(3 * time.Second)
		api.Set(context.Background(), "/ft/cluster/health/test_key", "panic", nil)

		time.Sleep(3 * time.Second)
		api.Set(context.Background(), "/ft/cluster/health/test_key", "don't panic", nil)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*7)

	success := false
	watcher.Watch(ctx, "/ft/cluster/health/test_key", func(val string) {
		if val == "panic" {
			panic("ahhhhh")
		}
		cancel()
		success = true
	})

	assert.True(t, success)
}
