package file

import (
	"os"
	"testing"
	"io/ioutil"
	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"path/filepath"
)

func TestSuccessfulInit(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, err := ioutil.TempFile(tempDir, "testFile")
	tempFile.Close()
	defer cleanupDir(tempDir)

	watcher, err := NewFileWatcher([]string{tempDir})

	assert.NoError(t, err)
	assert.NotNil(t, watcher)
}

func TestFailedInitFolderDoesntExist(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "testDir")
	defer cleanupDir(tempDir)

	_, err = NewFileWatcher([]string{tempDir, "dir-which-doesnt-exist"})

	assert.Error(t, err)
}

func TestFailedInitNoFolders(t *testing.T) {
	_, err := NewFileWatcher([]string{})

	assert.Error(t, err)
}

func TestFailedInitEmptyFolders(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "testDir")
	defer cleanupDir(tempDir)
	_, err = NewFileWatcher([]string{tempDir})

	assert.Error(t, err)
	assert.NotNil(t, err.Error())
}

func TestSuccessfulRead(t *testing.T) {
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

func TestFailedReadFileDoesntExist(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, err := ioutil.TempFile(tempDir, "read-environments")
	tempFile.WriteString("env1:url,env2:url")
	tempFile.Close()
	defer cleanupDir(tempDir)

	watcher, err := NewFileWatcher([]string{tempDir})
	_, err = Watcher(watcher).Read("this-file-doesnt-exist.fail")

	assert.Error(t, err)
}

func cleanupDir(tempDir string) {
	err := os.RemoveAll(tempDir)
	if err != nil {
		logrus.WithError(err).Error("Cannot remove temp dir", tempDir)
	}
}
