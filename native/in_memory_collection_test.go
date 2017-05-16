package native

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInMemoryIterator(t *testing.T) {
	it := &InMemoryUUIDCollection{collection: "collection"}

	values := []string{"hi", "my name is", "what?"}
	for _, v := range values {
		it.append(v)
	}

	assert.Equal(t, 3, it.Length())
	assert.False(t, it.Done())

	finished, val, err := it.Next()
	assert.NoError(t, err)
	assert.False(t, finished)
	assert.Equal(t, "hi", val)

	assert.Equal(t, 2, it.Length())
	assert.False(t, it.Done())

	finished, val, err = it.Next()
	assert.NoError(t, err)
	assert.False(t, finished)
	assert.Equal(t, "my name is", val)

	finished, val, err = it.Next()
	assert.NoError(t, err)
	assert.False(t, finished)
	assert.Equal(t, "what?", val)

	finished, _, err = it.Next()
	assert.NoError(t, err)
	assert.True(t, finished)

	err = it.Close() // no-op
	assert.NoError(t, err)
}

func TestLoadIntoMemory(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"my name", "is", "who?"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)

	it, err := LoadIntoMemory(uuidCollection, "collection", 0, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 3, it.Length())
}

func TestLoadIntoMemoryWithSkip(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"my name", "is", "who?"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)

	it, err := LoadIntoMemory(uuidCollection, "collection", 1, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 2, it.Length())

	done, val, err := it.Next()
	assert.False(t, done)
	assert.Equal(t, "is", val)
	assert.NoError(t, err)
}

func TestLoadIntoMemoryIgnoresBlanks(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"my name", "is", ""}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)

	it, err := LoadIntoMemory(uuidCollection, "collection", 1, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 1, it.Length())

	done, val, err := it.Next()
	assert.False(t, done)
	assert.Equal(t, "is", val)
	assert.NoError(t, err)
}

func TestLoadIntoMemoryBlacklisted(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"my name", "is", "who?"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(nil)

	it, err := LoadIntoMemory(uuidCollection, "collection", 0, func(uuid string) (bool, error) {
		if uuid == "my name" {
			return true, nil
		}
		return false, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, it.Length())

	done, val, err := it.Next()
	assert.False(t, done)
	assert.Equal(t, "is", val)
	assert.NoError(t, err)
}

func TestLoadIntoMemoryErrors(t *testing.T) {
	uuidCollection := &MockUUIDCollection{uuids: []string{"my name", "is", "who?"}}
	uuidCollection.On("Close").Return(nil)
	uuidCollection.On("Next").Return(errors.New("oh dear"))

	_, err := LoadIntoMemory(uuidCollection, "collection", 0, noopBlacklist)
	assert.Error(t, err)
}
