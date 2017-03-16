package native

import (
	"errors"
	"testing"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewNativeUUIDCollection(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	iter := &mgo.Iter{}

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDs", testCollection, 0).Return(iter, 11234, nil)

	actual, err := NewNativeUUIDCollection(mockDb, testCollection, 0)
	assert.NoError(t, err)
	assert.Equal(t, iter, actual.(*NativeUUIDCollection).iter)
	assert.Equal(t, 11234, actual.Length())

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionOpenFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	mockDb.On("Open").Return(mockTx, errors.New("fail"))

	_, err := NewNativeUUIDCollection(mockDb, testCollection, 0)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionFindFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	iter := &mgo.Iter{}

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDs", testCollection, 0).Return(iter, 11234, errors.New("fail"))

	_, err := NewNativeUUIDCollection(mockDb, testCollection, 0)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNewNativeUUIDCollectionForTimeWindow(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	iter := &mgo.Iter{}

	end := time.Now()
	start := end.Add(time.Minute * -1)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDsInTimeWindow", testCollection, start, end).Return(iter, 11234, nil)

	actual, err := NewNativeUUIDCollectionForTimeWindow(mockDb, testCollection, start, end)
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

	_, err := NewNativeUUIDCollectionForTimeWindow(mockDb, testCollection, start, end)
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

	iter := &mgo.Iter{}

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("FindUUIDsInTimeWindow", testCollection, start, end).Return(iter, 11234, errors.New("fail"))

	_, err := NewNativeUUIDCollectionForTimeWindow(mockDb, testCollection, start, end)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNativeUUIDCollection(t *testing.T) {
	db := startMongo(t)
	db.Open()
	defer db.Close()

	testUUID := uuid.New()
	insertTestContent(t, db, testUUID)

	uuidCollection, err := NewNativeUUIDCollection(db, "methode", 0)
	assert.NoError(t, err)

	found := false
	for !uuidCollection.Done() {
		if uuidCollection.Next() == testUUID {
			found = true
		}
	}

	assert.True(t, found)
	err = uuidCollection.Close()
	assert.NoError(t, err)

	cleanupTestContent(t, db, testUUID)
}
