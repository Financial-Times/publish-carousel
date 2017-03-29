package blacklist

import (
	"testing"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/stretchr/testify/assert"
)

func TestFileBasedBlacklist(t *testing.T) {
	blacklist, err := NewBuilder().FileBasedBlacklist("./test_blacklist.txt").Build()
	assert.NoError(t, err)

	content := &native.Content{}
	valid, err := blacklist.ValidForPublish("335a60b8-3092-11e0-9de3-00144feabdc0", content)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestFileBasedBlacklistValidUUID(t *testing.T) {
	blacklist, err := NewBuilder().FileBasedBlacklist("./test_blacklist.txt").Build()
	assert.NoError(t, err)

	content := &native.Content{}
	valid, err := blacklist.ValidForPublish("435a60b8-3092-11e0-9de3-00144feabdc0", content)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestFileNotFound(t *testing.T) {
	_, err := NewBuilder().FileBasedBlacklist("./not-a-real-file.txt").Build()
	assert.Error(t, err)
}
