package native

import (
	"time"

	"github.com/Sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
)

var expectedConnections = 1
var connections = 0

type Content struct {
	Body        map[string]interface{} `json:"content",bson:"content"`
	ContentType string                 `json:"contentType",bson:"content-type"`
}

// DB contains database functions
type DB interface {
	Open() (TX, error)
	Close()
}

// TX contains database transaction functions
type TX interface {
	ReadNativeContent(collectionId string, uuid string) (Content, error)
	FindUUIDsInTimeWindow(collectionId string, start time.Time, end time.Time) (*mgo.Iter, int, error)
	FindUUIDs(collectionId string) (*mgo.Iter, int, error)
	Ping() error
	Close()
}

// MongoTX wraps a mongo session
type MongoTX struct {
	session *mgo.Session
}

// MongoDB wraps a mango mongo session
type MongoDB struct {
	Urls       string
	Timeout    int
	MaxLimit   int
	CacheDelay int
	session    *mgo.Session
}

func (db *MongoDB) Open() (TX, error) {
	if db.session == nil {
		session, err := mgo.DialWithTimeout(db.Urls, time.Duration(db.Timeout)*time.Millisecond)
		if err != nil {
			return nil, err
		}
		db.session = session
		connections++

		if connections > expectedConnections {
			logrus.Warnf("There are more MongoDB connections opened than expected! Are you sure this is what you want? Open connections: %v, expected %v.", connections, expectedConnections)
		}
	}

	return &MongoTX{db.session.Copy()}, nil
}

// FindUUIDsInTimeWindow queries mongo for a list of uuids and returns an iterator
func (tx *MongoTX) FindUUIDsInTimeWindow(collectionID string, start time.Time, end time.Time) (*mgo.Iter, int, error) {
	collection := tx.session.DB("native-store").C(collectionID)

	query, projection := findUUIDsForTimeWindow(start, end)
	find := collection.Find(query).Select(projection)

	length, err := find.Count()
	return find.Iter(), length, err
}

func (tx *MongoTX) FindUUIDs(collectionID string) (*mgo.Iter, int, error) {
	collection := tx.session.DB("native-store").C(collectionID)

	query, projection := findUUIDs()
	find := collection.Find(query).Select(projection)

	length, err := find.Count()
	return find.Iter(), length, err
}

// ReadNativeContent queries mongo for a uuid and returns the native document
func (tx *MongoTX) ReadNativeContent(collectionID string, uuid string) (Content, error) {
	collection := tx.session.DB("native-store").C(collectionID)

	query := readNativeContentQuery(uuid)
	find := collection.Find(query)

	results := Content{}

	err := find.All(&results)

	if err != nil {
		return results, err
	}

	return results, nil
}

// Ping returns a mongo ping response
func (tx *MongoTX) Ping() error {
	return tx.session.Ping()
}

// Close closes the transaction
func (tx *MongoTX) Close() {
	tx.session.Close()
}

// Close closes the entire database connection
func (db *MongoDB) Close() {
	db.session.Close()
}
