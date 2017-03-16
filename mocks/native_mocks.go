package mocks

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/stretchr/testify/mock"
	mgo "gopkg.in/mgo.v2"
)

type MockDB struct {
	*mock.Mock
}

type MockTX struct {
	*mock.Mock
}

func (m *MockDB) Open() (native.TX, error) {
	args := m.Called()
	return args.Get(0).(native.TX), args.Error(1)
}

func (m *MockDB) Close() {
	m.Called()
}

func (t *MockTX) ReadNativeContent(collectionID string, uuid string) (*native.Content, error) {
	args := t.Called(collectionID, uuid)
	return args.Get(0).(*native.Content), args.Error(1)
}

func (t *MockTX) FindUUIDsInTimeWindow(collectionID string, start time.Time, end time.Time) (*mgo.Iter, int, error) {
	args := t.Called(collectionID, start, end)
	return args.Get(0).(*mgo.Iter), args.Int(1), args.Error(2)
}

func (t *MockTX) FindUUIDs(collectionID string, skip int) (*mgo.Iter, int, error) {
	args := t.Called(collectionID, skip)
	return args.Get(0).(*mgo.Iter), args.Int(1), args.Error(2)
}

func (t *MockTX) Ping() error {
	args := t.Called()
	return args.Error(0)
}

func (t *MockTX) Close() {
	t.Called()
}

func startMongo(t *testing.T) native.DB {
	if testing.Short() {
		t.Skip("Mongo integration for long tests only.")
	}

	mongoURL := os.Getenv("MONGO_TEST_URL")
	if strings.TrimSpace(mongoURL) == "" {
		t.Fatal("Please set the environment variable MONGO_TEST_URL to run mongo integration tests (e.g. MONGO_TEST_URL=localhost:27017). Alternatively, run `go test -short` to skip them.")
	}

	return native.NewMongoDatabase(mongoURL, 30000)
}
