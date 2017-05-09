package native

type Reader interface {
	Get(collection string, uuid string) (*Content, error)
}

type MongoReader struct {
	mongo DB
}

func NewMongoNativeReader(mongo DB) Reader {
	return &MongoReader{mongo}
}

func (m *MongoReader) Get(collection string, uuid string) (*Content, error) {
	tx, err := m.mongo.Open()

	if err != nil {
		return nil, err
	}

	defer tx.Close()

	content, err := tx.ReadNativeContent(collection, uuid)
	if err != nil {
		return nil, err
	}

	return content, nil
}
