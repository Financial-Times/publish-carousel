package native

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/stretchr/testify/assert"
)

func TestReadNativeContentQuery(t *testing.T) {
	testUUID := `e7743707-b4c4-4ab3-8043-bcb5f7fdd56b`
	query := readNativeContentQuery(testUUID)

	data, err := bson.MarshalJSON(query)
	assert.NoError(t, err)
	assert.Equal(t, `{"uuid":{"$binary":"53Q3B7TESrOAQ7y19/3Vaw==","$type":"0x4"}}`, strings.TrimSpace(string(data)))
}

func TestFindUUIDs(t *testing.T) {
	query, projection := findUUIDs()
	assert.Equal(t, bson.M{}, query)
	assert.Equal(t, contentUUIDProjection, projection)
}

func TestFindUUIDsForTimeWindow(t *testing.T) {
	end := time.Date(2017, 03, 16, 0, 0, 0, 0, time.UTC)
	start := end.Add(time.Minute * -1)

	query, projection := findUUIDsForTimeWindow(start, end)

	data, err := bson.MarshalJSON(query)
	assert.NoError(t, err)
	assert.Equal(t, `{"$and":[{"content.lastModified":{"$gte":"2017-03-15T23:59:00Z"}},{"content.lastModified":{"$lt":"2017-03-16T00:00:00Z"}}]}`, strings.TrimSpace(string(data)))
	assert.Equal(t, contentUUIDProjection, projection)
}
