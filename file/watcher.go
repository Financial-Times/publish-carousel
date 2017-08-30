package file

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"os"
	"strings"
	"io/ioutil"
	"github.com/pkg/errors"
	"fmt"
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
	log.WithField("folders", folders).Info("Reading file listing from given folders.")
	if len(folders) == 0 {
		return nil, errors.New("No folders were provided!")
	}

	paths := make(map[string]string)
	for _, folder := range folders {
		err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error walking file in folder [%s], or the folder itself.", folder))
			}
			if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
				paths[info.Name()] = path
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(paths) == 0 {
		return nil, errors.New("Didn't find any files to watch!")
	}

	log.WithField("filePaths", paths).Info("Collected list of files we can watch.")
	return &watcher{paths}, nil
}

func (e *watcher) Read(fileName string) (string, error) {
	path := e.filePaths[fileName]
	if path == "" {
		return "", errors.Errorf("File [%s] doesn't exist!", fileName)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "Error reading contents of file [%s]", fileName)
	}
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
