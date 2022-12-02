package s3

import (
	"bytes"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockS3API struct {
	mock.Mock
	s3iface.S3API
}

func (_m *MockS3API) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	ret := _m.Called(input)
	r0 := ret.Get(0).(*s3.PutObjectOutput)
	r1 := ret.Error(1)
	return r0, r1
}

func (_m *MockS3API) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	ret := _m.Called(input)
	r0 := ret.Get(0).(*s3.GetObjectOutput)
	r1 := ret.Error(1)
	return r0, r1
}

func (_m *MockS3API) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	ret := _m.Called(input)
	r0 := ret.Get(0).(*s3.ListObjectsOutput)
	r1 := ret.Error(1)
	return r0, r1
}
func TestWrite(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}

	expected := &s3.PutObjectInput{
		Bucket:      aws.String("test"),
		Key:         aws.String("fake-id/fake-key"),
		Body:        bytes.NewReader([]byte(`hi`)),
		ContentType: aws.String("application/json"),
	}

	output := &s3.PutObjectOutput{}

	mockS3.On("PutObject", mock.MatchedBy(func(putObj *s3.PutObjectInput) bool {
		assert.Equal(t, *expected.Bucket, *putObj.Bucket)
		assert.Equal(t, *expected.Key, *putObj.Key)

		actual, err := ioutil.ReadAll(putObj.Body)
		assert.NoError(t, err)

		expectedBody, err := ioutil.ReadAll(expected.Body)
		assert.NoError(t, err)
		assert.Equal(t, string(expectedBody), string(actual))
		assert.Equal(t, *expected.ContentType, *putObj.ContentType)
		return true
	})).Return(output, nil)
	err := rw.Write("fake-id", "fake-key", []byte(`hi`), "application/json")
	assert.NoError(t, err)
}

func TestWriteFails(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}

	expected := &s3.PutObjectInput{
		Bucket:      aws.String("test"),
		Key:         aws.String("fake-id/fake-key"),
		Body:        bytes.NewReader([]byte(`hi`)),
		ContentType: aws.String("application/json"),
	}

	output := &s3.PutObjectOutput{}

	mockS3.On("PutObject", mock.MatchedBy(func(putObj *s3.PutObjectInput) bool {
		assert.Equal(t, *expected.Bucket, *putObj.Bucket)
		assert.Equal(t, *expected.Key, *putObj.Key)

		actual, err := ioutil.ReadAll(putObj.Body)
		assert.NoError(t, err)

		expectedBody, err := ioutil.ReadAll(expected.Body)
		assert.NoError(t, err)
		assert.Equal(t, string(expectedBody), string(actual))
		assert.Equal(t, *expected.ContentType, *putObj.ContentType)
		return true
	})).Return(output, errors.New("oh no"))
	err := rw.Write("fake-id", "fake-key", []byte(`hi`), "application/json")
	assert.Error(t, err)
}

func TestRead(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}

	expected := &s3.GetObjectInput{
		Bucket: aws.String("test"),
		Key:    aws.String("key"),
	}

	output := &s3.GetObjectOutput{
		Body:        ioutil.NopCloser(strings.NewReader(`hi`)),
		ContentType: aws.String("application/json"),
	}

	mockS3.On("GetObject", mock.MatchedBy(func(obj *s3.GetObjectInput) bool {
		assert.Equal(t, *expected.Bucket, *obj.Bucket)
		assert.Equal(t, *expected.Key, *obj.Key)
		return true
	})).Return(output, nil)

	found, body, contentType, err := rw.Read("key")
	assert.NoError(t, err)
	assert.True(t, found)

	actual, err := ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, "hi", string(actual))
	assert.Equal(t, "application/json", *contentType)
}

func TestReadFails(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}

	expected := &s3.GetObjectInput{
		Bucket: aws.String("test"),
		Key:    aws.String("key"),
	}

	output := &s3.GetObjectOutput{
		Body:        ioutil.NopCloser(strings.NewReader(`hi`)),
		ContentType: aws.String("application/json"),
	}
	mockS3.On("GetObject", mock.MatchedBy(func(obj *s3.GetObjectInput) bool {
		assert.Equal(t, *expected.Bucket, *obj.Bucket)
		assert.Equal(t, *expected.Key, *obj.Key)
		return true
	})).Return(output, errors.New("fail"))
	_, _, _, err := rw.Read("key")
	assert.Error(t, err)
}

func TestReadKeyNotFound(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}

	expected := &s3.GetObjectInput{
		Bucket: aws.String("test"),
		Key:    aws.String("key"),
	}

	output := &s3.GetObjectOutput{
		Body:        ioutil.NopCloser(strings.NewReader(`hi`)),
		ContentType: aws.String("application/json"),
	}

	mockS3.On("GetObject", mock.MatchedBy(func(obj *s3.GetObjectInput) bool {
		assert.Equal(t, *expected.Bucket, *obj.Bucket)
		assert.Equal(t, *expected.Key, *obj.Key)
		return true
	})).Return(output, awserr.New("NoSuchKey", "fail", nil))

	found, _, _, err := rw.Read("key")
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestPing(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}
	err := rw.Ping()
	assert.NoError(t, err)

	rw = DefaultReadWriter{bucketName: "test", session: nil, lock: &sync.Mutex{}}
	err = rw.Ping()
	assert.Error(t, err)
}

func TestGetLatestKeyForID(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}

	expected := &s3.ListObjectsInput{
		Bucket: aws.String("test"),
		Prefix: aws.String("fake-id/"),
	}

	var contents []*s3.Object
	contents = append(contents, &s3.Object{Key: aws.String("fake-key"), LastModified: aws.Time(time.Now().Add(-1 * time.Second))})
	contents = append(contents, &s3.Object{Key: aws.String("fake-key2"), LastModified: aws.Time(time.Now().Add(-1 * time.Minute))})
	contents = append(contents, &s3.Object{Key: aws.String("fake-key3"), LastModified: aws.Time(time.Now())})

	output := &s3.ListObjectsOutput{
		Contents: contents,
	}

	mockS3.On("ListObjects", mock.MatchedBy(func(obj *s3.ListObjectsInput) bool {
		assert.Equal(t, *expected.Bucket, *obj.Bucket)
		assert.Equal(t, *expected.Prefix, *obj.Prefix)
		return true
	})).Return(output, nil)

	key, err := rw.GetLatestKeyForID("fake-id")
	assert.NoError(t, err)
	assert.Equal(t, "fake-key3", key)
}

func TestGetLatestKeyForIDFails(t *testing.T) {
	mockS3 := new(MockS3API)
	rw := DefaultReadWriter{bucketName: "test", session: mockS3, lock: &sync.Mutex{}}

	expected := &s3.ListObjectsInput{
		Bucket: aws.String("test"),
		Prefix: aws.String("fake-id/"),
	}

	output := &s3.ListObjectsOutput{}

	mockS3.On("ListObjects", mock.MatchedBy(func(obj *s3.ListObjectsInput) bool {
		assert.Equal(t, *expected.Bucket, *obj.Bucket)
		assert.Equal(t, *expected.Prefix, *obj.Prefix)
		return true
	})).Return(output, errors.New("hi"))

	_, err := rw.GetLatestKeyForID("fake-id")
	assert.Error(t, err)
}

func TestNewRW(t *testing.T) {
	rw := NewReadWriter("region", "bucketName").(*DefaultReadWriter)
	assert.Equal(t, "bucketName", rw.bucketName)
	assert.Equal(t, "region", *rw.config.Region)
	assert.NotNil(t, rw.config.HTTPClient)
	assert.NotNil(t, rw.lock)
}
