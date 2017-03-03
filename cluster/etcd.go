package cluster

import (
	"context"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/proxy"
)

// Watcher see Watch func for details
type Watcher interface {
	Watch(key string, callback func(val string))
}

type etcdWatcher struct {
	api etcd.KeysAPI
}

// NewEtcdWatcher returns a new etcd watcher
func NewEtcdWatcher(connection string) (Watcher, error) {
	transport := &http.Transport{
		Dial: proxy.Direct.Dial,
		ResponseHeaderTimeout: 10 * time.Second,
		MaxIdleConnsPerHost:   100,
	}

	etcdCfg := etcd.Config{
		Endpoints:               strings.Split(connection, ","),
		Transport:               transport,
		HeaderTimeoutPerRequest: 10 * time.Second,
	}

	etcdClient, err := etcd.New(etcdCfg)
	if err != nil {
		log.Printf("Cannot load etcd configuration: [%v]", err)
		return nil, err
	}

	api := etcd.NewKeysAPI(etcdClient)
	return &etcdWatcher{api}, nil
}

// Watch starts an etcd watch on a given key, and triggers the callback when found
func (e *etcdWatcher) Watch(key string, callback func(val string)) {
	watcher := e.api.Watcher(key, &etcd.WatcherOptions{AfterIndex: 0, Recursive: false})

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
