package native

import "encoding/json"

type Reader interface {
	Get(collection string, uuid string) (*Content, string, error) // TODO the second parameter is the hash of the content
}

type MongoReader struct {
	mongo DB
}

func NewMongoNativeReader(mongo DB) Reader {
	return &MongoReader{mongo}
}

func (m *MongoReader) Get(collection string, uuid string) (*Content, string, error) {
	tx, err := m.mongo.Open()

	if err != nil {
		return nil, "", err
	}

	defer tx.Close()

	content, err := tx.ReadNativeContent(collection, uuid)
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

	return content, hash, nil
}
