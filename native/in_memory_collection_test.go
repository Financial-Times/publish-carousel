package native

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInMemoryIterator(t *testing.T) {
	it := &InMemoryUUIDCollection{collection: "collection"}

	values := []string{"1", "2", "3"}
	for _, v := range values {
		it.append(v)
	}

	assert.Equal(t, 3, it.Length())
	assert.False(t, it.Done())

	finished, val, err := it.Next()
	assert.NoError(t, err)
	assert.False(t, finished)
	assert.Equal(t, "1", val)

	assert.Equal(t, 2, it.Length())
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

	it, err := LoadIntoMemory(uuidCollection, "collection", 0, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 3, it.Length())
}

func TestLoadIntoMemoryWithSkip(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(3)

	it, err := LoadIntoMemory(uuidCollection, "collection", 1, noopBlacklist)
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

	it, err := LoadIntoMemory(uuidCollection, "collection", 1, noopBlacklist)
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

	it, err := LoadIntoMemory(uuidCollection, "collection", 0, func(uuid string) (bool, error) {
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

func TestLoadIntoMemoryErrors(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"1", "2", "3"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(errors.New("oh dear"))
	uuidCollection.On("Length").Return(3)

	_, err := LoadIntoMemory(uuidCollection, "collection", 0, noopBlacklist)
	assert.Error(t, err)
}

func TestLoadIntoMemoryEmptyCollection(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)
	uuidCollection.On("Length").Return(0)

	it, err := LoadIntoMemory(uuidCollection, "collection", 0, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 0, it.Length())
}
