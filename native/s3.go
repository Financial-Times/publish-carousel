package native

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Financial-Times/publish-carousel/s3"
)

const persistedUUIDsSuffix = "-uuids"

func persistInS3(rw s3.ReadWriter, collection *InMemoryUUIDCollection) error {
	key := time.Now().UTC().Format(`20060102T15040599`) + ".json"

	b, err := json.Marshal(collection.uuids)
	if err != nil {
		return err
	}

	return rw.Write(collection.collection+persistedUUIDsSuffix, key, b, "application/json")
}

func readFromS3(rw s3.ReadWriter, collection string) ([]string, error) {
	key, err := rw.GetLatestKeyForID(collection + persistedUUIDsSuffix)
	if err != nil {
		return nil, err
	}

	found, data, contentType, err := rw.Read(key)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, errors.New("Key not found, has it recently been deleted?")
	}

	if contentType == nil || *contentType != "application/json" {
		return nil, errors.New("Unexpected or nil content type")
	}

	dec := json.NewDecoder(data)
	var uuids []string
	err = dec.Decode(&uuids)

	if err != nil {
		return nil, err
	}

	return uuids, nil
}
