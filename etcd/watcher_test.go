package etcd

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	etcdClient "github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
)

func startEtcd(t *testing.T) (Watcher, error) {
	if testing.Short() {
		t.Skip("Skipping etcd integration test")
	}

	etcdURL := os.Getenv("ETCD_TEST_URL")
	if strings.TrimSpace(etcdURL) == "" {
		t.Fatal("Please set the environment variable ETCD_TEST_URL to run etcd integration tests (e.g. ETCD_TEST_URL=http://localhost:2379). Alternatively, run tests with `-short` to skip them.")
	}

	return NewEtcdWatcher([]string{etcdURL})
}

func TestWatch(t *testing.T) {
	watcher, err := startEtcd(t)
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
	watcher, err := startEtcd(t)
	assert.NoError(t, err)

	cfg := etcdClient.Config{
		Endpoints: []string{"http://localhost:2379"},
	}

	client, err := etcdClient.New(cfg)
	assert.NoError(t, err)

	api := etcdClient.NewKeysAPI(client)
	go func() {
		time.Sleep(1 * time.Second)
		api.Set(context.Background(), "/ft/cluster/health/test_key", "panic", nil)

		time.Sleep(1 * time.Second)
		api.Set(context.Background(), "/ft/cluster/health/test_key", "don't panic", nil)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

	success := false
	watcher.Watch(ctx, "/ft/cluster/health/test_key", func(val string) {
		if val == "panic" {
			panic("ahhhhh")
		}
		success = true
		cancel()
	})

	assert.True(t, success)
}
