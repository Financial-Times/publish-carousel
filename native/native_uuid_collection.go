package native

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"gopkg.in/mgo.v2/bson"
)

const mongoCursorTimeout = 10 * time.Minute
const maxBatchSize = 80

type UUIDCollection interface {
	io.Closer
	Next() (bool, string, error)
	Length() int
	Done() bool
}

type NativeUUIDCollection struct {
	collection string
	iter       DBIter
	length     int
}

type NativeUUIDCollectionBuilder struct {
	db            DB
	inMemory      *InMemoryCollectionBuilder
	isBlacklisted blacklist.IsBlacklisted
}

func NewNativeUUIDCollectionBuilder(mongo DB, rw s3.ReadWriter, isBlacklisted blacklist.IsBlacklisted) *NativeUUIDCollectionBuilder {
	return &NativeUUIDCollectionBuilder{db: mongo, isBlacklisted: isBlacklisted, inMemory: NewInMemoryCollectionBuilder(rw)}
}

func (b *NativeUUIDCollectionBuilder) NewNativeUUIDCollectionForTimeWindow(collection string, start time.Time, end time.Time, maximumThrottle time.Duration) (UUIDCollection, error) {
	tx, err := b.db.Open()
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

	if size > maxBatchSize {
		return maxBatchSize, nil
	}

	log.WithField("batch", int(size-1)).Info("Computed batch size for cursor.")
	return int(size - 1), nil
}

func (b *NativeUUIDCollectionBuilder) NewNativeUUIDCollection(ctx context.Context, collection string, skip int) (UUIDCollection, error) {
	tx, err := b.db.Open()
	if err != nil {
		return nil, err
	}

	iter, length, err := tx.FindUUIDs(collection, 0, 100)
	if err != nil {
		return nil, err
	}

	cursor := &NativeUUIDCollection{collection: collection, iter: iter, length: length}

	inMemory, err := b.inMemory.LoadIntoMemory(ctx, cursor, collection, skip, b.isBlacklisted)
	return inMemory, err
}

func (n *NativeUUIDCollection) Next() (bool, string, error) {
	result := map[string]interface{}{}

	success := n.iter.Next(&result)

	if !success || n.iter.Err() != nil {
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
