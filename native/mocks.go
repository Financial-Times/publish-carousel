package native

import (
	"time"

	"github.com/stretchr/testify/mock"
)

type MockDB struct {
	mock.Mock
}

type MockTX struct {
	mock.Mock
}

func (m *MockDB) Open() (TX, error) {
	args := m.Called()
	return args.Get(0).(TX), args.Error(1)
}

func (m *MockDB) Close() {
	m.Called()
}

func (t *MockTX) ReadNativeContent(collectionID string, uuid string) (*Content, error) {
	args := t.Called(collectionID, uuid)
	return args.Get(0).(*Content), args.Error(1)
}

func (t *MockTX) FindUUIDsInTimeWindow(collectionID string, start time.Time, end time.Time, batchsize int) (DBIter, int, error) {
	args := t.Called(collectionID, start, end, batchsize)
	return args.Get(0).(DBIter), args.Int(1), args.Error(2)
}

func (t *MockTX) FindUUIDs(collectionID string, skip int, batchsize int) (DBIter, int, error) {
	args := t.Called(collectionID, skip, batchsize)
	return args.Get(0).(DBIter), args.Int(1), args.Error(2)
}

func (t *MockTX) Ping() error {
	args := t.Called()
	return args.Error(0)
}

func (t *MockTX) Close() {
	t.Called()
}

type MockReader struct {
	mock.Mock
}

func (m *MockReader) Get(collection string, uuid string) (*Content, string, error) {
	args := m.Called(collection, uuid)
	return args.Get(0).(*Content), args.String(1), args.Error(2)
}

type MockDBIter struct {
	mock.Mock
}

func (m *MockDBIter) Done() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDBIter) Next(result interface{}) bool {
	args := m.Called(result)
	return args.Bool(0)
}

func (m *MockDBIter) Err() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDBIter) Timeout() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDBIter) Close() error {
	args := m.Called()
	return args.Error(0)
}
