package s3

import (
	"io"

	"github.com/stretchr/testify/mock"
)

type MockReadWriter struct {
	mock.Mock
}

func (m *MockReadWriter) Write(id string, key string, b []byte, contentType string) error {
	args := m.Called(id, key, b, contentType)
	return args.Error(0)
}

func (m *MockReadWriter) Read(key string) (bool, io.ReadCloser, *string, error) {
	args := m.Called(key)
	return args.Bool(0), args.Get(1).(io.ReadCloser), args.Get(2).(*string), args.Error(3)
}

func (m *MockReadWriter) GetLatestKeyForID(id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockReadWriter) Ping() error {
	args := m.Called()
	return args.Error(0)
}
