package blacklist

import (
	"errors"
	"strings"

	"github.com/Financial-Times/publish-carousel/native"
)

// FilterImages will filter image types based on the `type` json attribute
func (b *Builder) FilterImages() *Builder {
	b.chain = append(b.chain, imageFilter())
	return b
}

func imageFilter() blacklistFilter {
	return func(uuid string, content *native.Content) (bool, error) {
		if content.Body == nil {
			return false, errors.New("no body found")
		}

		contentType, ok := content.Body["type"]
		if !ok {
			return true, nil
		}

		return strings.ToLower(contentType.(string)) != "image", nil
	}
}
