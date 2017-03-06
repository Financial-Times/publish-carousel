package cms

import (
	"github.com/Financial-Times/publish-carousel/native"
	log "github.com/Sirupsen/logrus"
)

type Notifier interface {
	Notify(content native.Content, hash string) error
}

type cmsNotifier struct {
}

func NewNotifier() Notifier {
	return &cmsNotifier{}
}

func (c *cmsNotifier) Notify(content native.Content, hash string) error {
	log.Info(content.ContentType)
	log.Info(hash)
	return nil
}
