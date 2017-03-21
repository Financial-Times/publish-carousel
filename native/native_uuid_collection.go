package native

import (
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pborman/uuid"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const mongoCursorTimeout = 10 * time.Minute

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

func NewNativeUUIDCollectionForTimeWindow(mongo DB, collection string, start time.Time, end time.Time, maximumThrottle time.Duration) (UUIDCollection, error) {
	tx, err := mongo.Open()
	if err != nil {
		return nil, err
	}

	batchsize, err := computeBatchsize(maximumThrottle)
	if err != nil {
		return nil, err
	}

	iter, length, err := tx.FindUUIDsInTimeWindow(collection, start, end, batchsize)
	if err != nil {
		return nil, err
	}

	return &NativeUUIDCollection{collection: collection, iter: iter, length: length}, nil
}

// This computes the batch size to use for the mongo cursor. We need to ensure the cursor does not timeout server side during the cycle
func computeBatchsize(interval time.Duration) (int, error) {
	if interval >= mongoCursorTimeout {
		return -1, fmt.Errorf("Cannot have an interval greater than the mongo default timeout. Interval %v, mongo timeout %v", interval.String(), mongoCursorTimeout.String())
	}

	size := mongoCursorTimeout.Nanoseconds() / interval.Nanoseconds()
	if size <= 1 {
		return 1, nil
	}

	logrus.WithField("batch", int(size-1)).Info("Computed batch size for cursor.")
	return int(size - 1), nil
}

func NewNativeUUIDCollection(mongo DB, collection string, skip int, maximumThrottle time.Duration) (UUIDCollection, error) {
	tx, err := mongo.Open()
	if err != nil {
		return nil, err
	}

	batchsize, err := computeBatchsize(maximumThrottle)
	if err != nil {
		return nil, err
	}

	iter, length, err := tx.FindUUIDs(collection, skip, batchsize)
	if err != nil {
		return nil, err
	}

	return &NativeUUIDCollection{collection: collection, iter: iter, length: length}, nil
}

func (n *NativeUUIDCollection) Next() (bool, string, error) {
	result := map[string]interface{}{}

	success := n.iter.Next(&result)

	if !success && n.iter.Err() != nil {
		return true, "", n.iter.Err()
	}

	if n.iter.Timeout() {
		return true, "", errors.New("Mongo timeout detected")
	}

	if !success {
		return true, "", nil
	}

	val, ok := result["uuid"]
	if !ok {
		return false, "", nil // this document has no uuid
	}

	return false, parseBinaryUUID(val), nil
}

func parseBinaryUUID(bin interface{}) string {
	return uuid.UUID(bin.(bson.Binary).Data).String()
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
