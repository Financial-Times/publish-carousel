package native

import (
	"errors"
	"time"

	mgo "gopkg.in/mgo.v2"
)

type UUIDCollection interface {
	Next() (bool, string, error)
	Length() int
	Done() bool
	Close() error
}

type NativeUUIDCollection struct {
	collection string
	iter       *mgo.Iter
	length     int
}

type contentUUID struct {
	UUID string `json:"uuid" bson:"uuid"`
}

func NewNativeUUIDCollectionForTimeWindow(mongo DB, collection string, start time.Time, end time.Time) (UUIDCollection, error) {
	tx, err := mongo.Open()
	if err != nil {
		return nil, err
	}

	iter, length, err := tx.FindUUIDsInTimeWindow(collection, start, end)
	if err != nil {
		return nil, err
	}

	return &NativeUUIDCollection{collection: collection, iter: iter, length: length}, nil
}

func NewNativeUUIDCollection(mongo DB, collection string, skip int) (UUIDCollection, error) {
	tx, err := mongo.Open()
	if err != nil {
		return nil, err
	}

	iter, length, err := tx.FindUUIDs(collection, skip)
	if err != nil {
		return nil, err
	}

	return &NativeUUIDCollection{collection: collection, iter: iter, length: length}, nil
}

func (n *NativeUUIDCollection) Next() (bool, string, error) {
	result := struct {
		Content contentUUID `bson:"content"`
	}{}
	success := n.iter.Next(&result)

	if !success && n.iter.Err() != nil {
		return true, "", n.iter.Err()
	}

	if n.iter.Timeout() {
		return true, result.Content.UUID, errors.New("Mongo timeout detected")
	}

	if !success {
		return true, "", nil
	}

	return false, result.Content.UUID, nil
}

func (n *NativeUUIDCollection) Close() error {
	return n.iter.Close()
}

func (n *NativeUUIDCollection) Length() int {
	return n.length
}

func (n *NativeUUIDCollection) Done() bool {
	return n.iter.Done()
}
