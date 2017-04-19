package blacklist

import (
	"testing"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/stretchr/testify/assert"
)

func TestFileBasedBlacklist(t *testing.T) {
	blacklist, err := NewBuilder().FileBasedBlacklist("./test_blacklist.txt").Build()
	assert.NoError(t, err)

	uuids := map[string]bool{
		"335a60b8-3092-11e0-9de3-00144feabdc0": false,
		"271f1e94-cd71-11df-9c82-00144feab49a": false,
		"399f1746-f1ae-49c1-a633-b0875a035372": true,
		"2f34db47-687f-499e-a4c0-8fced650ba25": true,
		"002c88c6-cd6c-11df-ab20-00144feab49a": false,
		"4fce28d4-2401-4c17-b484-29da67386cba": true,
	}

	content := &native.Content{}
	for uuid, expectedValid := range uuids {
		actualValid, err := blacklist.ValidForPublish(uuid, content)
		assert.NoError(t, err)
		assert.Equal(t, expectedValid, actualValid, "The validation should match")
	}
}

func TestFileNotFound(t *testing.T) {
	_, err := NewBuilder().FileBasedBlacklist("./not-a-real-file.txt").Build()
	assert.Error(t, err)
}
