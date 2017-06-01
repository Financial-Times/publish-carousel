package native

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var noopBlacklist = func(uuid string) (bool, error) { return false, nil }

func TestComputeBatchSize(t *testing.T) {
	tests := []struct {
		expected int
		duration time.Duration
		err      bool
	}{
		{
			expected: 9,
			duration: 1 * time.Minute,
			err:      false,
		},
		{
			expected: 39,
			duration: 15 * time.Second,
			err:      false,
		},
		{
			expected: maxBatchSize,
			duration: 2 * time.Second,
			err:      false,
		},
		{
			expected: 1,
			duration: 9 * time.Minute,
			err:      false,
		},
		{
			duration: 10 * time.Minute,
			err:      true,
		},
		{
			duration: 1 * time.Hour,
			err:      true,
		},
	}

	for _, test := range tests {
		batch, err := computeBatchsize(test.duration)
		if test.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, batch)
		}
	}
}

func TestNewNativeUUIDCollection(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	iter := new(MockDBIter)

	iter.On("Next", mock.AnythingOfType("*map[string]interface {}")).Return(false, "", nil)
	iter.On("Close").Return(nil)
	iter.On("Timeout").Return(false)
	iter.On("Err").Return(nil)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDs", testCollection, 0, 100).Return(iter, 11234, nil)

	actual, err := NewNativeUUIDCollection(context.Background(), mockDb, testCollection, 0, noopBlacklist)
	assert.NoError(t, err)
	assert.Equal(t, 0, actual.Length())

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionOpenFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	mockDb.On("Open").Return(mockTx, errors.New("fail"))

	_, err := NewNativeUUIDCollection(context.Background(), mockDb, testCollection, 0, noopBlacklist)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionFindFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	iter := new(MockDBIter)
	iter.On("Next").Return(true, "", nil)
	iter.On("Close").Return(nil)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDs", testCollection, 0, 100).Return(iter, 11234, errors.New("fail"))

	_, err := NewNativeUUIDCollection(context.Background(), mockDb, testCollection, 0, noopBlacklist)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionForTimeWindow(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	iter := new(MockDBIter)

	end := time.Now()
	start := end.Add(time.Minute * -1)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDsInTimeWindow", testCollection, start, end, 9).Return(iter, 11234, nil)

	actual, err := NewNativeUUIDCollectionForTimeWindow(mockDb, testCollection, start, end, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, iter, actual.(*NativeUUIDCollection).iter)
	assert.Equal(t, 11234, actual.Length())

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionForTimeWindowOpenFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	end := time.Now()
	start := end.Add(time.Minute * -1)

	mockDb.On("Open").Return(mockTx, errors.New("fail"))

	_, err := NewNativeUUIDCollectionForTimeWindow(mockDb, testCollection, start, end, time.Minute)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionForTimeWindowFindFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	end := time.Now()
	start := end.Add(time.Minute * -1)

	iter := new(MockDBIter)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDsInTimeWindow", testCollection, start, end, 9).Return(iter, 11234, errors.New("fail"))

	_, err := NewNativeUUIDCollectionForTimeWindow(mockDb, testCollection, start, end, time.Minute)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNativeUUIDCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("mongo integration test")
	}

	db := startMongo(t)
	db.Open()
	defer db.Close()

	testUUID := uuid.New()
	insertTestContent(t, db, testUUID, time.Now())

	t.Log(testUUID)

	uuidCollection, err := NewNativeUUIDCollection(context.Background(), db, "methode", 0, noopBlacklist)
	assert.NoError(t, err)

	found := false
	for !uuidCollection.Done() {
		_, result, errNext := uuidCollection.Next()
		assert.NoError(t, errNext)

		if result == testUUID {
			found = true
		}
	}

	assert.True(t, found)
	err = uuidCollection.Close()
	assert.NoError(t, err)

	cleanupTestContent(t, db, testUUID)
}
