package native

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInMemoryIterator(t *testing.T) {
	it := &InMemoryUUIDCollection{collection: "collection"}

	values := []string{"1", "2", "3"}
	for _, v := range values {
		it.append(v)
	}

	assert.Len(t, it.uuids, 3)
	assert.False(t, it.Done())

	finished, val, err := it.Next()
	assert.NoError(t, err)
	assert.False(t, finished)
	assert.Equal(t, "1", val)

	assert.Len(t, it.uuids, 2)
	assert.False(t, it.Done())

	finished, val, err = it.Next()
	assert.NoError(t, err)
	assert.False(t, finished)
	assert.Equal(t, "2", val)

	finished, val, err = it.Next()
	assert.NoError(t, err)
	assert.False(t, finished)
	assert.Equal(t, "3", val)

	finished, _, err = it.Next()
	assert.NoError(t, err)
	assert.True(t, finished)

	err = it.Close() // no-op
	assert.NoError(t, err)
}

func TestLoadIntoMemory(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(3)

	builder := &InMemoryCollectionBuilder{nil}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 0, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 3, it.Length())
}

func TestLoadIntoMemoryWithSkip(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(3)

	builder := &InMemoryCollectionBuilder{nil}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 1, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 3, it.Length())

	done, val, err := it.Next()
	assert.False(t, done)
	assert.Equal(t, "2", val)
	assert.NoError(t, err)
}

func TestLoadIntoMemoryIgnoresBlanks(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", " "}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(3)

	builder := &InMemoryCollectionBuilder{nil}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 1, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 2, it.Length())

	done, val, err := it.Next()
	assert.False(t, done)
	assert.Equal(t, "2", val)
	assert.NoError(t, err)
}

func TestLoadIntoMemoryBlacklisted(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(3)

	builder := &InMemoryCollectionBuilder{nil}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 0, func(uuid string) (bool, error) {
		if uuid == "1" {
			return true, nil
		}
		return false, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, it.Length())

	done, val, err := it.Next()
	assert.False(t, done)
	assert.Equal(t, "2", val)
	assert.NoError(t, err)
}

func TestLoadIntoMemoryWithContextInterrupt(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil).Run(func(arg1 mock.Arguments) {
		time.Sleep(time.Millisecond * 500)
	})
	uuidCollection.On("Length").Return(3)

	ctx, cancel := context.WithCancel(context.Background())

	wg := &sync.WaitGroup{}
	wg.Add(1)

	completed := false

	go func() {
		defer wg.Done()
		builder := &InMemoryCollectionBuilder{nil}
		_, err := builder.LoadIntoMemory(ctx, uuidCollection, "collection", 0, noopBlacklist)
		assert.NoError(t, err)

		completed = true
	}()

	cancel()
	wg.Wait()
	assert.True(t, completed)
}

func TestLoadIntoMemoryErrors(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(errors.New("oh dear"))
	uuidCollection.On("Length").Return(3)

	builder := &InMemoryCollectionBuilder{nil}
	_, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 0, noopBlacklist)
	assert.Error(t, err)
}

func TestLoadIntoMemoryEmptyCollection(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(0)

	builder := &InMemoryCollectionBuilder{nil}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 0, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 0, it.Length())
}

func TestPersistToS3DuringInMemoryStartup(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(3)

	expectedJSON, _ := json.Marshal(uuidCollection.uuids)

	mockS3RW := new(s3.MockReadWriter)
	mockS3RW.On("Write", "collection-uuids", mock.AnythingOfType("string"), expectedJSON, "application/json").Return(nil)

	builder := &InMemoryCollectionBuilder{mockS3RW}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 0, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 3, it.Length())
}

func TestReadFromS3DuringInMemoryStartup(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{}}
	uuidCollection.On("Close").Return(nil)

	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	contentType := "application/json"

	rw.On("Read", "key").Return(true, ioutil.NopCloser(strings.NewReader(`["1","2","3","4"]`)), &contentType, nil)

	builder := &InMemoryCollectionBuilder{rw}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 1, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 4, it.Length())

	rw.AssertExpectations(t)
	uuidCollection.AssertExpectations(t)
}

func TestReadFromS3DuringInMemoryStartupInvalidSkip(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(1)

	expectedJSON, _ := json.Marshal(uuidCollection.uuids)

	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)
	rw.On("Write", "collection-uuids", mock.AnythingOfType("string"), expectedJSON, "application/json").Return(nil)

	contentType := "application/json"

	rw.On("Read", "key").Return(true, ioutil.NopCloser(strings.NewReader(`["1","2","3","4"]`)), &contentType, nil)

	builder := &InMemoryCollectionBuilder{rw}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 9, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 1, it.Length())

	rw.AssertExpectations(t)
	uuidCollection.AssertExpectations(t)
}

func TestReadFromS3DuringInMemoryStartupFails(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(3)

	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)
	rw.On("Write", "collection-uuids", mock.AnythingOfType("string"), []byte(`["2","3"]`), "application/json").Return(nil)

	contentType := "application/json"
	rw.On("Read", "key").Return(false, ioutil.NopCloser(strings.NewReader("")), &contentType, errors.New("no s3 for you"))

	builder := &InMemoryCollectionBuilder{rw}

	it, err := builder.LoadIntoMemory(context.Background(), uuidCollection, "collection", 1, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 3, it.Length())

	rw.AssertExpectations(t)
	uuidCollection.AssertExpectations(t)
}
