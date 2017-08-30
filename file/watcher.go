package file

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"os"
	"strings"
	"io/ioutil"
)

// Watcher see Watch func for details
type Watcher interface {
	Watch(ctx context.Context, key string, callback func(val string))
	Read(fileName string) (string, error)
}

type watcher struct {
	filePaths map[string]string
}

// NewFileWatcher returns a new file watcher
func NewFileWatcher(folders []string) (Watcher, error) {
	paths := make(map[string]string)
	for _, folder := range folders {
		filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
				paths[info.Name()] = path
			}
			return nil
		})
	}
	log.WithField("filepaths", paths).Info("Gathered filepaths.")
	return &watcher{paths}, nil
}

func (e *watcher) Read(fileName string) (string, error) {
	data, _ := ioutil.ReadFile(e.filePaths[fileName])
	return string(data), nil
}

// Watch starts an etcd watch on a given key, and triggers the callback when found
func (e *watcher) Watch(ctx context.Context, key string, callback func(val string)) {
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
