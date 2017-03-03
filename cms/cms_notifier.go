package cms

import "github.com/Financial-Times/publish-carousel/native"

type Notifier interface {
	Notify(content native.Content, hash string) error
}
