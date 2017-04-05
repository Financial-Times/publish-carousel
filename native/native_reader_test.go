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

	actual, hash, err := reader.Get(testCollection, testUUID)
	assert.NoError(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)

	assert.Equal(t, "5cdd15a873608087be07a41b7f1a04e96d3a66fe7a9b0faac71f8d05", hash)
	assert.Equal(t, "application/vnd.expect-this", actual.ContentType)
}

func TestNativeReaderMongoOpenFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	testUUID := "fake-uuid"

	reader := NewMongoNativeReader(mockDb)

	mockDb.On("Open").Return(mockTx, errors.New("mongo broke mate"))

	_, _, err := reader.Get(testCollection, testUUID)
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

	_, _, err := reader.Get(testCollection, testUUID)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestNativeReaderJSONMarshalFails(t *testing.T) {
	mockDb := new(MockDB)
	mockTx := new(MockTX)

	testCollection := "testing-123"
	testUUID := "fake-uuid"

	testBody := make(map[string]interface{})
	testBody["errrr"] = func() {}
	testContent := Content{Body: testBody, ContentType: "application/vnd.expect-this"}

	reader := NewMongoNativeReader(mockDb)

	mockDb.On("Open").Return(mockTx, nil)
	mockTx.On("Close")
	mockTx.On("ReadNativeContent", testCollection, testUUID).Return(&testContent, nil)

	_, _, err := reader.Get(testCollection, testUUID)
	assert.Error(t, err)

	mockDb.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}
