package cms

type Notifier interface {
	Notify(content map[string]interface{}, hash string) error
}
