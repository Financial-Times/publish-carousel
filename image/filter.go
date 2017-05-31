package image

import (
	"errors"
	"strings"

	"github.com/Financial-Times/publish-carousel/native"
)

// NoOpImageFilter always returns false; useful for testing
var NoOpImageFilter = func(uuid string, content *native.Content) (bool, error) { return false, nil }

// Filter given a uuid and native content object from mongo, will test whether the content is an image, or error if there is no body or content is nil
type Filter func(uuid string, content *native.Content) (bool, error)

// NewFilter returns a function which will return true if the native content is an image.
func NewFilter() Filter {
	return func(uuid string, content *native.Content) (bool, error) {
		if content == nil || content.Body == nil {
			return false, errors.New("no body found")
		}

		contentType, ok := content.Body["type"]
		if !ok {
			return false, nil
		}

		return strings.ToLower(contentType.(string)) == "image", nil
	}
}
