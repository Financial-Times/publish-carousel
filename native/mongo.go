package native

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
)

var expectedConnections = 1
var connections = 0

type Content struct {
	Body        map[string]interface{} `bson:"content"`
	ContentType string                 `bson:"content-type"`
}

// DB contains database functions
type DB interface {
	Open() (TX, error)
	Close()
}

// TX contains database transaction functions
type TX interface {
	ReadNativeContent(collectionId string, uuid string) (*Content, error)
	FindUUIDsInTimeWindow(collectionId string, start time.Time, end time.Time, batchsize int) (DBIter, int, error)
	FindUUIDs(collectionId string, skip int, batchsize int) (DBIter, int, error)
	Ping() error
	Close()
}

// MongoTX wraps a mongo session
type MongoTX struct {
	session *mgo.Session
}

// MongoDB wraps a mango mongo session
type MongoDB struct {
	Urls    string
	Timeout int
	lock    *sync.Mutex
	session *mgo.Session
}

func NewMongoDatabase(connection string, timeout int) DB {
	return &MongoDB{Urls: connection, Timeout: timeout, lock: &sync.Mutex{}}
}

func (db *MongoDB) Open() (TX, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.session == nil {
		session, err := mgo.DialWithTimeout(db.Urls, time.Duration(db.Timeout)*time.Millisecond)
		if err != nil {
			log.WithError(err).Error("Session error")
			return nil, err
		}

		db.session = session
		connections++

		if connections > expectedConnections {
			log.Warnf("There are more MongoDB connections opened than expected! Are you sure this is what you want? Open connections: %v, expected %v.", connections, expectedConnections)
		}
	}

	return &MongoTX{db.session.Copy()}, nil
}

// FindUUIDsInTimeWindow queries mongo for a list of uuids and returns an iterator
func (tx *MongoTX) FindUUIDsInTimeWindow(collectionID string, start time.Time, end time.Time, batchsize int) (DBIter, int, error) {
	collection := tx.session.DB("native-store").C(collectionID)

	query, projection := findUUIDsForTimeWindowQueryElements(start, end)
	find := collection.Find(query).Select(projection).Batch(batchsize)

	count, err := find.Count()
	return find.Iter(), count, err
}

//returns all uuids for a collection sorted by lastodified date, if no lastmodified exists records are returned at the end of the list
func (tx *MongoTX) FindUUIDs(collectionID string, skip int, batchsize int) (DBIter, int, error) {
	collection := tx.session.DB("native-store").C(collectionID)

	query, projection, sortByDate := findUUIDsQueryElements()
	find := collection.Find(query).Select(projection).Sort(sortByDate).Batch(batchsize)

	if skip > 0 {
		find.Skip(skip)
	}

	count, err := find.Count()
	return find.Iter(), count + skip, err // add count to skip as this correctly computes the total size of the cursor
}

// ReadNativeContent queries mongo for a uuid and returns the native document
func (tx *MongoTX) ReadNativeContent(collectionID string, uuid string) (*Content, error) {
	collection := tx.session.DB("native-store").C(collectionID)

	query := readNativeContentQuery(uuid)
	find := collection.Find(query)

	result := &Content{}
	err := find.One(result)

	if err != nil {
		return result, err
	}

	return result, nil
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

type DBIter interface {
	Done() bool
	Next(result interface{}) bool
	Err() error
	Timeout() bool
	Close() error
}
