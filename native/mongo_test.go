package native

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

func startMongo(t *testing.T) DB {
	if testing.Short() {
		t.Skip("Mongo integration for long tests only.")
	}

	mongoURL := os.Getenv("MONGO_TEST_URL")
	if strings.TrimSpace(mongoURL) == "" {
		t.Fatal("Please set the environment variable MONGO_TEST_URL to run mongo integration tests (e.g. MONGO_TEST_URL=localhost:27017). Alternatively, run `go test -short` to skip them.")
	}

	return NewMongoDatabase(mongoURL, 30000)
}

func TestCreateDB(t *testing.T) {
	db := NewMongoDatabase("test-url", 30000)
	mongo := db.(*MongoDB)
	assert.Equal(t, "test-url", mongo.Urls)
	assert.Equal(t, 30000, mongo.Timeout)
	assert.NotNil(t, mongo.lock)
}

func insertTestContent(t *testing.T, mongo DB, testUUID string, lastModified time.Time) {
	testContent := make(map[string]interface{})

	props := make(map[string]interface{})
	props["uuid"] = testUUID
	props["publishReference"] = "tid_" + testUUID
	props["lastModified"] = lastModified.UTC().Format(time.RFC3339)

	testContent["content"] = props
	testContent["uuid"] = bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(testUUID))}

	session := mongo.(*MongoDB).session.Copy()
	defer session.Close()

	err := session.DB("native-store").C("methode").Insert(testContent)
	assert.NoError(t, err)
}

func cleanupTestContent(t *testing.T, mongo DB, testUUIDs ...string) {
	session := mongo.(*MongoDB).session.Copy()
	defer session.Close()
	for _, testUUID := range testUUIDs {
		err := session.DB("native-store").C("methode").Remove(bson.M{"content.uuid": testUUID})
		assert.NoError(t, err)
	}
}

func TestFindByUUID(t *testing.T) {
	db := startMongo(t)
	defer db.Close()

	tx, err := db.Open()
	defer tx.Close()
	assert.NoError(t, err)

	testUUID := uuid.NewUUID().String()
	t.Log("Test uuid to use", testUUID)
	insertTestContent(t, db, testUUID, time.Now())

	iter, count, err := tx.FindUUIDs("methode", 0, 10)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, count)

	found := false
	for !iter.Done() {
		result := map[string]interface{}{}
		iter.Next(&result)

		t.Log(result)
		val, ok := result["uuid"]
		if !ok {
			continue
		}

		if parseBinaryUUID(val) == testUUID {
			found = true
		}
	}

	assert.True(t, found)
	cleanupTestContent(t, db, testUUID)
}

func TestFindUUIDsDateSort(t *testing.T) {
	db := startMongo(t)
	defer db.Close()

	tx, err := db.Open()
	defer tx.Close()
	assert.NoError(t, err)

	testUUID1 := uuid.NewUUID().String()
	testUUID2 := uuid.NewUUID().String()
	testUUID3 := uuid.NewUUID().String()
	testUUIDs := []string{testUUID1, testUUID2, testUUID3}

	insertTestContent(t, db, testUUID2, time.Now().Add(-10 * time.Second))
	insertTestContent(t, db, testUUID1, time.Now())
	insertTestContent(t, db, testUUID3, time.Now().Add(-20 * time.Second))

	iter, count, err := tx.FindUUIDs("methode", 0, 10)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, count)
	actualUUIDs := []string{}
	for !iter.Done() {
		result := map[string]interface{}{}
		iter.Next(&result)
		val, ok := result["uuid"]
		actualUUIDs = append(actualUUIDs, parseBinaryUUID(val))
		if !ok {
			continue
		}
	}
	assert.Equal(t, testUUIDs, actualUUIDs, "uuids do not match therefore they are not in expected descending date order")

	cleanupTestContent(t, db, testUUIDs...)
}

func TestFindByTimeWindow(t *testing.T) {
	db := startMongo(t)
	defer db.Close()

	tx, err := db.Open()
	defer tx.Close()
	assert.NoError(t, err)

	testUUID := uuid.NewUUID().String()
	testUUID2 := uuid.NewUUID().String()
	t.Log("Test uuids to use", testUUID, testUUID2)

	insertTestContent(t, db, testUUID, time.Now().Add(time.Second * -1))
	insertTestContent(t, db, testUUID2, time.Now().Add(time.Minute * -2))

	end := time.Now()
	start := end.Add(time.Minute * -1)

	iter, count, err := tx.FindUUIDsInTimeWindow("methode", start, end, 10)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, count)

	found := false
	for !iter.Done() {
		result := map[string]interface{}{}
		iter.Next(&result)

		t.Log(result)
		if parseBinaryUUID(result["uuid"]) == testUUID {
			found = true
		}

		if parseBinaryUUID(result["uuid"]) == testUUID2 {
			t.Log("Should not find this uuid as it is outside the window.")
			t.Fail()
		}
	}

	assert.True(t, found)
	cleanupTestContent(t, db, testUUID, testUUID2)
}

func TestReadNativeContent(t *testing.T) {
	db := startMongo(t)
	defer db.Close()

	tx, err := db.Open()
	defer tx.Close()
	assert.NoError(t, err)

	testUUID := uuid.NewUUID().String()
	t.Log("Test uuid to use", testUUID)
	insertTestContent(t, db, testUUID, time.Now())

	content, err := tx.ReadNativeContent("methode", testUUID)
	assert.NoError(t, err)
	assert.NotNil(t, content)

	assert.Equal(t, testUUID, content.Body["uuid"])
	assert.Equal(t, "tid_" + testUUID, content.Body["publishReference"])
	cleanupTestContent(t, db, testUUID)
}

func TestPing(t *testing.T) {
	db := startMongo(t)
	defer db.Close()

	tx, err := db.Open()
	defer tx.Close()
	assert.NoError(t, err)

	err = tx.Ping()
	assert.NoError(t, err)
}

func TestTransactionCloses(t *testing.T) {
	db := startMongo(t)
	defer db.Close()

	tx, err := db.Open()
	assert.NoError(t, err)

	tx.Close()
	assert.Panics(t, func() {
		tx.(*MongoTX).session.Ping()
	})
}

func TestDBCloses(t *testing.T) {
	db := startMongo(t)
	tx, err := db.Open()
	assert.NoError(t, err)

	tx.Close()
	db.Close()
	assert.Panics(t, func() {
		db.(*MongoDB).session.Ping()
	})
}

func TestCheckMongoURLsValidUrls(t *testing.T) {
	err := CheckMongoURLs("valid-url.com:1234", 1)
	assert.NoError(t, err)
}

func TestCheckMongoURLsMissingUrls(t *testing.T) {
	err := CheckMongoURLs("", 1)
	assert.Error(t, err)
}

func TestCheckMongoURLsSmallerNumberOfUrls(t *testing.T) {
	err := CheckMongoURLs("valid-url.com:1234", 2)
	assert.Error(t, err)
}

func TestCheckMongoURLsGreaterNumberOfUrls(t *testing.T) {
	err := CheckMongoURLs("valid-url.com:1234,second-valid-url.com:1234", 1)
	assert.NoError(t, err)
}

func TestCheckMongoURLsMissingPort(t *testing.T) {
	err := CheckMongoURLs("valid-url.com:", 1)
	assert.Error(t, err)
}

func TestCheckMongoURLsMissingHost(t *testing.T) {
	err := CheckMongoURLs(":1234", 1)
	assert.Error(t, err)
}
