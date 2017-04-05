package native

import (
	"time"

	"github.com/pborman/uuid"

	"gopkg.in/mgo.v2/bson"
)

func readNativeContentQuery(nativeUUID string) bson.M {
	return bson.M{"uuid": bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(nativeUUID))}}
}

var uuidProjection = bson.M{
	"uuid": 1,
}

func findUUIDsForTimeWindow(start time.Time, end time.Time) (bson.M, bson.M) {
	query := bson.M{
		"$and": []bson.M{
			{
				"content.lastModified": bson.M{
					"$gte": start.UTC().Format(time.RFC3339),
				},
			},
			{
				"content.lastModified": bson.M{
					"$lt": end.UTC().Format(time.RFC3339),
				},
			},
		},
	}

	return query, uuidProjection
}

func findUUIDs() (bson.M, bson.M) {
	return bson.M{}, uuidProjection
}
