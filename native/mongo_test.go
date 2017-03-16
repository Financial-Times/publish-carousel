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

func insertTestContent(t *testing.T, mongo DB, testUUID string) {
	testContent := make(map[string]interface{})

	props := make(map[string]interface{})
	props["uuid"] = testUUID
	props["publishReference"] = "tid_" + testUUID
	props["lastModified"] = time.Now().Format(time.RFC3339Nano)

	testContent["content"] = props
	testContent["uuid"] = bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(testUUID))}

	session := mongo.(*MongoDB).session.Copy()

	err := session.DB("native-store").C("methode").Insert(testContent)
	assert.NoError(t, err)
}

func cleanupTestContent(t *testing.T, mongo DB, testUUID string) {
	session := mongo.(*MongoDB).session.Copy()

	err := session.DB("native-store").C("methode").Remove(bson.M{"content.uuid": testUUID})
	assert.NoError(t, err)
}

func TestFindByUUID(t *testing.T) {
	db := startMongo(t)

	tx, err := db.Open()
	assert.NoError(t, err)

	testUUID := uuid.NewUUID().String()
	t.Log("Test uuid to use", testUUID)
	insertTestContent(t, db, testUUID)

	iter, count, err := tx.FindUUIDs("methode", 0)
	assert.NotEqual(t, 0, count)
	assert.NoError(t, err)

	found := false
	for !iter.Done() {
		result := struct {
			Content contentUUID `bson:"content"`
		}{}
		iter.Next(&result)

		t.Log(result.Content.UUID)
		if result.Content.UUID == testUUID {
			found = true
		}
	}

	assert.True(t, found)
	cleanupTestContent(t, db, testUUID)
}

func TestFindByTimeWindow(t *testing.T) {
	db := startMongo(t)

	tx, err := db.Open()
	assert.NoError(t, err)

	testUUID := uuid.NewUUID().String()
	t.Log("Test uuid to use", testUUID)

	end := time.Now()
	start := end.Add(time.Minute * -1)
	insertTestContent(t, db, testUUID)

	iter, count, err := tx.FindUUIDsInTimeWindow("methode", start, end)
	assert.NotEqual(t, 0, count)
	assert.NoError(t, err)

	found := false
	for !iter.Done() {
		result := struct {
			Content contentUUID `bson:"content"`
		}{}
		iter.Next(&result)

		t.Log(result.Content.UUID)
		if result.Content.UUID == testUUID {
			found = true
		}
	}

	assert.True(t, found)
	cleanupTestContent(t, db, testUUID)
}

func TestPing(t *testing.T) {

}
