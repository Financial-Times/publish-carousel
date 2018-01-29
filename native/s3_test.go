package native

import (
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistToS3(t *testing.T) {
	rw := new(s3.MockReadWriter)
	cursor := &InMemoryUUIDCollection{collection: "collection", uuids: make([]string, 0)}

	rw.On("Write", "collection-uuids", mock.AnythingOfType("string"), []byte("[]"), "application/json").Return(nil)

	err := persistInS3(rw, cursor)
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, rw)
}

func TestPersistToS3Fails(t *testing.T) {
	rw := new(s3.MockReadWriter)
	cursor := &InMemoryUUIDCollection{collection: "collection", uuids: make([]string, 0)}

	rw.On("Write", "collection-uuids", mock.AnythingOfType("string"), []byte("[]"), "application/json").Return(errors.New("oh no"))

	err := persistInS3(rw, cursor)
	assert.Error(t, err)
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3(t *testing.T) {
	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	contentType := "application/json"

	rw.On("Read", "key").Return(true, ioutil.NopCloser(strings.NewReader(`["a-uuid"]`)), &contentType, nil)

	uuids, err := readFromS3(rw, "collection")
	assert.NotNil(t, uuids)
	assert.NoError(t, err)

	assert.Len(t, uuids, 1)
	assert.Equal(t, uuids[0], "a-uuid")
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3NoPreviousSave(t *testing.T) {
	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("", errors.New("nooo"))

	uuids, err := readFromS3(rw, "collection")
	assert.Nil(t, uuids)
	assert.EqualError(t, err, "nooo")
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3ReadFails(t *testing.T) {
	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	contentType := "application/json"
	rw.On("Read", "key").Return(false, ioutil.NopCloser(strings.NewReader("[]]")), &contentType, errors.New("something failed"))

	uuids, err := readFromS3(rw, "collection")
	assert.Nil(t, uuids)
	assert.EqualError(t, err, "something failed")
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3NotFound(t *testing.T) {
	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	contentType := "application/json"
	rw.On("Read", "key").Return(false, ioutil.NopCloser(strings.NewReader("[]]")), &contentType, nil)

	uuids, err := readFromS3(rw, "collection")
	assert.Nil(t, uuids)
	assert.EqualError(t, err, "Key not found, has it recently been deleted?")
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3UnsupportMediaType(t *testing.T) {
	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	contentType := "application/something-else"
	rw.On("Read", "key").Return(true, ioutil.NopCloser(strings.NewReader("[]]")), &contentType, nil)

	uuids, err := readFromS3(rw, "collection")
	assert.Nil(t, uuids)
	assert.EqualError(t, err, "Unexpected or nil content type")
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3NilMediaType(t *testing.T) {
	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	var contentType *string

	rw.On("Read", "key").Return(true, ioutil.NopCloser(strings.NewReader("[]]")), contentType, nil)

	uuids, err := readFromS3(rw, "collection")
	assert.Nil(t, uuids)
	assert.EqualError(t, err, "Unexpected or nil content type")
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3InvalidDataType(t *testing.T) {
	rw := new(s3.MockReadWriter)
	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	contentType := "application/json"

	rw.On("Read", "key").Return(true, ioutil.NopCloser(strings.NewReader("{}")), &contentType, nil)

	uuids, err := readFromS3(rw, "collection")
	assert.Nil(t, uuids)
	assert.EqualError(t, err, "json: cannot unmarshal object into Go value of type []string")
	mock.AssertExpectationsForObjects(t, rw)
}

func TestReadFromS3ClosesResponseBody(t *testing.T) {
	rw := new(s3.MockReadWriter)
	body := &cluster.MockBody{Reader: strings.NewReader(`{}}`)}

	body.On("Read")
	body.On("Close").Return(nil)

	rw.On("GetLatestKeyForID", "collection-uuids").Return("key", nil)

	contentType := "application/json"

	rw.On("Read", "key").Return(true, body, &contentType, nil)

	uuids, err := readFromS3(rw, "collection")
	assert.Nil(t, uuids)
	assert.EqualError(t, err, "json: cannot unmarshal object into Go value of type []string")

	mock.AssertExpectationsForObjects(t, rw, body)
}
