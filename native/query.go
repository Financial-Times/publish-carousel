package native

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

func readNativeContentQuery(uuid string) bson.M {
	return bson.M{"uuid": bson.Binary{Kind: 0x04, Data: []byte(uuid)}}
}

func findUUIDsForTimeWindow(start time.Time, end time.Time) (bson.M, bson.M) {
	query := bson.M{
		"$and": []bson.M{
			{
				"content.lastModified": bson.M{
					"$gte": start.Format(time.RFC3339),
				},
			},
			{
				"content.lastModified": bson.M{
					"$lt": end.Format(time.RFC3339),
				},
			},
		},
	}

	projection := bson.M{
		"content.uuid": 1,
	}

	return query, projection
}
