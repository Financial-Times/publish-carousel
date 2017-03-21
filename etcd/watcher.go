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
	Watch(key string, callback func(val string))
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

// Watch starts an etcd watch on a given key, and triggers the callback when found
func (e *etcdWatcher) Watch(key string, callback func(val string)) {
	watcher := e.api.Watcher(key, &etcdClient.WatcherOptions{AfterIndex: 0, Recursive: false})

	for {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			log.Info("Error waiting for change under %v in etcd. %v\n Sleeping 10s...", key, err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		if resp != nil && resp.Node != nil {
			callback(resp.Node.Value)
		}
	}
}
