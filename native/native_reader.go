package native

type Reader interface {
	Get(uuid string) (map[string]interface{}, string, error) // TODO the second parameter is the hash of the content
}
