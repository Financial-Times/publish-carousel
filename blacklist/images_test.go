package blacklist

import (
	"testing"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/stretchr/testify/assert"
)

func TestFilterImages(t *testing.T) {
	blacklist, err := NewBuilder().FilterImages().Build()
	assert.NoError(t, err)

	body := make(map[string]interface{})
	body["type"] = "Image"
	content := &native.Content{Body: body}

	valid, err := blacklist.ValidForPublish("fake-uuid", content)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestImageFilterAllowsContent(t *testing.T) {
	blacklist, err := NewBuilder().FilterImages().Build()
	assert.NoError(t, err)

	body := make(map[string]interface{})
	body["type"] = "Content"
	content := &native.Content{Body: body}

	valid, err := blacklist.ValidForPublish("fake-uuid", content)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestImageFilterAllowsNoType(t *testing.T) {
	blacklist, err := NewBuilder().FilterImages().Build()
	assert.NoError(t, err)

	body := make(map[string]interface{})
	content := &native.Content{Body: body}

	valid, err := blacklist.ValidForPublish("fake-uuid", content)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestImageFilterFailsMissingBody(t *testing.T) {
	blacklist, err := NewBuilder().FilterImages().Build()
	assert.NoError(t, err)

	content := &native.Content{}

	_, err = blacklist.ValidForPublish("fake-uuid", content)
	assert.Error(t, err)
}
