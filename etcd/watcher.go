package etcd

import (
	"context"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	etcdClient "github.com/coreos/etcd/client"
	"golang.org/x/net/proxy"
)

// Watcher see Watch func for details
type Watcher interface {
	Watch(ctx context.Context, key string, callback func(val string))
	Read(key string) (string, error)
}

type etcdWatcher struct {
	api etcdClient.KeysAPI
}

// NewEtcdWatcher returns a new etcd watcher
func NewEtcdWatcher(endpointsList []string) (Watcher, error) {
	transport := &http.Transport{
		Dial: proxy.Direct.Dial,
		ResponseHeaderTimeout: 10 * time.Second,
		MaxIdleConnsPerHost:   100,
	}

	etcdCfg := etcdClient.Config{
		Endpoints:               endpointsList,
		Transport:               transport,
		HeaderTimeoutPerRequest: 10 * time.Second,
	}

	client, err := etcdClient.New(etcdCfg)
	if err != nil {
		log.Printf("Cannot load etcd configuration: [%v]", err)
		return nil, err
	}

	api := etcdClient.NewKeysAPI(client)
	return &etcdWatcher{api}, nil
}

func (e *etcdWatcher) Read(key string) (string, error) {
	resp, err := e.api.Get(context.Background(), key, nil)
	if err != nil {
		return "", err
	}

	return resp.Node.Value, nil
}

// Watch starts an etcd watch on a given key, and triggers the callback when found
func (e *etcdWatcher) Watch(ctx context.Context, key string, callback func(val string)) {
	watcher := e.api.Watcher(key, &etcdClient.WatcherOptions{AfterIndex: 0, Recursive: false})

	for {
		if ctx.Err() != nil {
			log.WithField("key", key).Info("Etcd watcher canceled.")
			break
		}

		resp, err := watcher.Next(ctx)
		if err != nil {
			log.WithError(err).WithField("key", key).Info("Error occurred while waiting for change in etcd. Sleeping 10s")
			time.Sleep(10 * time.Second)
			continue
		}

		if resp != nil && resp.Node != nil {
			runCallback(resp, callback)
		}
	}
}

func runCallback(resp *etcdClient.Response, callback func(val string)) {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("panic", r).Error("Watcher callback panicked! This should not happen, and indicates there is a bug.")
		}
	}()
	callback(resp.Node.Value)
}
