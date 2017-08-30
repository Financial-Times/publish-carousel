package file

import (
	"os"
	"testing"
	"io/ioutil"
	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"context"
	"time"
)

const refreshInterval = time.Second * 30
const expectedValue = "env1:url,env2:url"

var expectedValues = []string{expectedValue, "env3:url,env4:url", "env5:url,env6:url"}

func TestSuccessfulInit(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, _ := ioutil.TempFile(tempDir, "testFile")
	tempFile.Close()
	defer cleanupDir(tempDir)

	watcher, err := NewFileWatcher([]string{tempDir}, refreshInterval)

	assert.NoError(t, err)
	assert.NotNil(t, watcher)
}

func TestFailedInitFolderDoesntExist(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "testDir")
	defer cleanupDir(tempDir)

	_, err := NewFileWatcher([]string{tempDir, "dir-which-doesnt-exist"}, refreshInterval)

	assert.Error(t, err)
}

func TestFailedInitNoFolders(t *testing.T) {
	_, err := NewFileWatcher([]string{}, refreshInterval)

	assert.Error(t, err)
}

func TestFailedInitEmptyFolders(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "testDir")
	defer cleanupDir(tempDir)
	_, err := NewFileWatcher([]string{tempDir}, refreshInterval)

	assert.Error(t, err)
	assert.NotNil(t, err.Error())
}

func TestSuccessfulRead(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, _ := ioutil.TempFile(tempDir, "read-environments")
	tempFile.WriteString(expectedValue)
	tempFile.Close()
	defer cleanupDir(tempDir)
	watcher, _ := NewFileWatcher([]string{tempDir}, refreshInterval)
	environments, err := Watcher(watcher).Read(filepath.Base(tempFile.Name()))

	assert.NoError(t, err)
	assert.Equal(t, expectedValue, environments)
}

func TestFailedReadFileDoesntExist(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, _ := ioutil.TempFile(tempDir, "read-environments")
	tempFile.WriteString(expectedValue)
	tempFile.Close()
	defer cleanupDir(tempDir)

	watcher, _ := NewFileWatcher([]string{tempDir}, refreshInterval)
	_, err := Watcher(watcher).Read("this-file-doesnt-exist.fail")

	assert.Error(t, err)
}

func TestSuccessfulWatch(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, _ := ioutil.TempFile(tempDir, "read-environments")
	tempFile.WriteString(expectedValues[0])
	tempFile.Close()
	defer cleanupDir(tempDir)
	watcher, _ := NewFileWatcher([]string{tempDir}, time.Second*1)

	go func() {
		time.Sleep(2 * time.Second)
		ioutil.WriteFile(tempFile.Name(), []byte(expectedValues[1]), 0600)
		tempFile.Sync()

		time.Sleep(2 * time.Second)
		ioutil.WriteFile(tempFile.Name(), []byte(expectedValues[2]), 0600)
		tempFile.Sync()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	i := 0
	watcher.Watch(ctx, filepath.Base(tempFile.Name()), func(val string) {
		assert.Equal(t, expectedValues[i], val)
		if i++; i == len(expectedValues) {
			cancel()
		}
	})
}

func TestWatchCallbackPanics(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, _ := ioutil.TempFile(tempDir, "read-environments")
	tempFile.WriteString(expectedValue)
	defer tempFile.Close()
	//defer cleanupDir(tempDir)
	watcher, _ := NewFileWatcher([]string{tempDir}, time.Second*1)

	go func() {
		time.Sleep(2 * time.Second)
		ioutil.WriteFile(tempFile.Name(), []byte("panic"), 0600)
		tempFile.Sync()

		time.Sleep(2 * time.Second)
		ioutil.WriteFile(tempFile.Name(), []byte("don't panic"), 0600)
		tempFile.Sync()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	success := false
	watcher.Watch(ctx, filepath.Base(tempFile.Name()), func(val string) {
		if val == "panic" {
			success = true
			panic("PANIC")
		}
		if val == "don't panic" {
			cancel()
		}
	})

	assert.True(t, success)
}

func cleanupDir(tempDir string) {
	err := os.RemoveAll(tempDir)
	if err != nil {
		logrus.WithError(err).Error("Cannot remove temp dir", tempDir)
	}
}
