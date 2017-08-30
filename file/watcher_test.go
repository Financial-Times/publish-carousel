package file

import (
	"os"
	"testing"
	"io/ioutil"
	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "testDir")
	tempFile, err := ioutil.TempFile(tempDir, "testFile")
	tempFile.Close()
	defer cleanupDir(tempDir)

	watcher, err := NewFileWatcher([]string{tempDir})

	assert.NoError(t, err)
	assert.NotNil(t, watcher)
}

func cleanupDir(tempDir string) {
	err := os.RemoveAll(tempDir)
	if err != nil {
		logrus.WithError(err).Error("Cannot remove temp dir", tempDir)
	}
}
