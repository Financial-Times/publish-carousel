package file

import (
	"os"
	"testing"
	"io/ioutil"
	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"path/filepath"
)

func TestWatchInit(t *testing.T) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, err := ioutil.TempFile(tempDir, "testFile")
	tempFile.Close()
	defer cleanupDir(tempDir)

	watcher, err := NewFileWatcher([]string{tempDir})

	assert.NoError(t, err)
	assert.NotNil(t, watcher)
}

func TestRead(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, err := ioutil.TempFile(tempDir, "read-environments")
	tempFile.WriteString("env1:url,env2:url")
	tempFile.Close()
	defer cleanupDir(tempDir)
	watcher, err := NewFileWatcher([]string{tempDir})
	environments, err := Watcher(watcher).Read(filepath.Base(tempFile.Name()))

	assert.NoError(t, err)
	assert.Equal(t, "env1:url,env2:url", environments)
}

func cleanupDir(tempDir string) {
	err := os.RemoveAll(tempDir)
	if err != nil {
		logrus.WithError(err).Error("Cannot remove temp dir", tempDir)
	}
}
