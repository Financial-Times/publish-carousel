package native

import (
	"encoding/json"
)

type Reader interface {
	Get(uuid string) (*Content, string, error) // TODO the second parameter is the hash of the content
}

type MongoReader struct {
	mongo      DB
	collection string
}

func (m *MongoReader) Get(uuid string) (*Content, string, error) {
	tx, err := m.mongo.Open()

	if err != nil {
		return nil, "", err
	}

	defer tx.Close()

	content, err := tx.ReadNativeContent(m.collection, uuid)
	if err != nil {
		return nil, "", err
	}

	data, err := json.Marshal(content.Body)
	if err != nil {
		return nil, "", err
	}

	hash, err := Hash(data)
	if err != nil {
		return nil, "", err
	}

	return &content, hash, nil
}
