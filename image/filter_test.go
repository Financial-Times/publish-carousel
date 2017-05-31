package image

import (
	"testing"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/stretchr/testify/assert"
)

func TestFilterImages(t *testing.T) {
	filter := NewFilter()

	body := make(map[string]interface{})
	body["type"] = "Image"
	content := &native.Content{Body: body}

	isImage, err := filter("fake-uuid", content)
	assert.NoError(t, err)
	assert.True(t, isImage)
}

func TestImageFilterAllowsContent(t *testing.T) {
	filter := NewFilter()

	body := make(map[string]interface{})
	body["type"] = "Content"
	content := &native.Content{Body: body}

	isImage, err := filter("fake-uuid", content)
	assert.NoError(t, err)
	assert.False(t, isImage)
}

func TestImageFilterAllowsNoType(t *testing.T) {
	filter := NewFilter()

	body := make(map[string]interface{})
	content := &native.Content{Body: body}

	isImage, err := filter("fake-uuid", content)
	assert.NoError(t, err)
	assert.False(t, isImage)
}

func TestImageFilterFailsMissingBody(t *testing.T) {
	filter := NewFilter()

	content := &native.Content{}

	_, err := filter("fake-uuid", content)
	assert.Error(t, err)
}

func TestImageFilterFailsNilContent(t *testing.T) {
	filter := NewFilter()

	_, err := filter("fake-uuid", nil)
	assert.Error(t, err)
}

// pointless test for 100% coverage (vanity)
func TestNoOp(t *testing.T) {
	isImage, err := NoOpImageFilter("", nil)
	assert.False(t, isImage)
	assert.NoError(t, err)
}
