package file

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"os"
	"strings"
)

// Watcher see Watch func for details
type Watcher interface {
	Watch(ctx context.Context, key string, callback func(val string))
	Read(key string) (string, error)
}

type fileWatcher struct {
	filePaths map[string]string
}

// NewFileWatcher returns a new etcd watcher
func NewFileWatcher(folders []string) (Watcher, error) {
	var filePaths map[string]string
	for _, folder := range folders {
		filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			log.Info("\tFile: [%s]", info.Name())
			if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
				filePaths[info.Name()] = path
			}
			return nil
		})
	}
	log.WithField("filepaths", filePaths).Info("Gathered filepaths.")
	return &fileWatcher{filePaths}, nil
}

func (e *fileWatcher) Read(key string) (string, error) {
	//TODO implement
	return "", nil
}

// Watch starts an etcd watch on a given key, and triggers the callback when found
func (e *fileWatcher) Watch(ctx context.Context, key string, callback func(val string)) {
	//TODO implement
}

//func runCallback(resp *etcdClient.Response, callback func(val string)) {
//	defer func() {
//		if r := recover(); r != nil {
//			log.WithField("panic", r).Error("Watcher callback panicked! This should not happen, and indicates there is a bug.")
//		}
//	}()
//	callback(resp.Node.Value)
//}
