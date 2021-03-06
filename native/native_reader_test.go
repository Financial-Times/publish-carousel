package native

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNativeReaderGet(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	testUUID := "fake-uuid"

	testContent := Content{Body: make(map[string]interface{}), ContentType: "application/vnd.expect-this"}

	reader := NewMongoNativeReader(mockDb)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("Close")
	mockTx.On("ReadNativeContent", testCollection, testUUID).Return(&testContent, nil)

	actual, err := reader.Get(testCollection, testUUID)
	assert.NoError(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)

	assert.Equal(t, "application/vnd.expect-this", actual.ContentType)
	assert.Equal(t, "", actual.OriginSystemID)
}

func TestNativeReaderGetWithOrigin(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	testUUID := "fake-uuid"

	testContent := Content{Body: make(map[string]interface{}), ContentType: "application/vnd.expect-this", OriginSystemID: "http://cmdb.ft.com/systems/cct"}

	reader := NewMongoNativeReader(mockDb)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("Close")
	mockTx.On("ReadNativeContent", testCollection, testUUID).Return(&testContent, nil)

	actual, err := reader.Get(testCollection, testUUID)
	assert.NoError(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)

	assert.Equal(t, "application/vnd.expect-this", actual.ContentType)
	assert.Equal(t, "http://cmdb.ft.com/systems/cct", actual.OriginSystemID)
}

func TestNativeReaderMongoOpenFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	testUUID := "fake-uuid"

	reader := NewMongoNativeReader(mockDb)

	mockDb.On("Open").Return(mockTx, errors.New("mongo broke mate"))

	_, err := reader.Get(testCollection, testUUID)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNativeReaderMongoReadFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	testUUID := "fake-uuid"

	reader := NewMongoNativeReader(mockDb)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("Close")
	mockTx.On("ReadNativeContent", testCollection, testUUID).Return(&Content{}, errors.New("couldn't find it for ya"))

	_, err := reader.Get(testCollection, testUUID)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}
