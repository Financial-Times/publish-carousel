package scheduler

import (
	"errors"
	"sync"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Important!
//
// The tests in this file can block indefinitely if the ThrottledWholeCollection cycle does not work as expected! Please be aware of failures due to tests timing out.
//

func TestWholeCollectionCycleRunWithMetadata(t *testing.T) {
	expectedUUID := uuid.NewUUID().String()
	expectedSkip := 500

	task := mockTask(expectedUUID, nil, nil)

	throttleCalled := make(chan struct{}, 1)
	opened := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)

	throttle := mockThrottle(time.Millisecond*50, throttleCalled)

	iter := mockIterWithCollectionSize(expectedUUID, 2000, closed)
	happyIter(iter)

	tx := mockTx(iter, nil)
	db := mockDB(opened, tx, nil)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	cycle := NewThrottledWholeCollectionCycle("name", uuidCollectionBuilder, "collection", "origin", time.Millisecond*50, throttle, task)

	metadata := CycleMetadata{Completed: expectedSkip, Iteration: 1, Attempts: 36}
	cycle.SetMetadata(metadata)

	cycle.Start()

	<-opened

	assert.Len(t, cycle.State(), 1)
	assert.Contains(t, cycle.State(), startingState)

	<-closed

	assert.Len(t, cycle.State(), 1)
	assert.Contains(t, cycle.State(), runningState)

	<-throttleCalled

	cycle.Stop()

	<-throttleCalled

	assert.Len(t, cycle.State(), 1)
	assert.Contains(t, cycle.State(), stoppedState)

	mock.AssertExpectationsForObjects(t, throttle, iter, tx, db, task)

	assert.Equal(t, 1, cycle.Metadata().Iteration)
	assert.Equal(t, 501, cycle.Metadata().Completed)
	assert.Equal(t, 37, cycle.Metadata().Attempts)
}

func TestWholeCollectionCycleTaskPrepareFails(t *testing.T) {
	expectedUUID := uuid.NewUUID().String()

	task := mockTask(expectedUUID, errors.New("i fail soz"), nil)

	throttleCalled := make(chan struct{}, 1)
	opened := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)

	throttle := mockThrottle(time.Millisecond*50, throttleCalled)

	iter := mockIterWithCollectionSize(expectedUUID, 2000, closed)
	happyIter(iter)

	tx := mockTx(iter, nil)
	db := mockDB(opened, tx, nil)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("name", uuidCollectionBuilder, "collection", "origin", time.Millisecond*50, throttle, task)

	c.Start()

	<-opened
	<-closed

	<-throttleCalled

	c.Stop()

	assert.Len(t, c.State(), 1)
	assert.Contains(t, c.State(), stoppedState)

	mock.AssertExpectationsForObjects(t, throttle, iter, tx, db, task)
	assert.Equal(t, 1, c.Metadata().Errors)
}

func TestWholeCollectionCycleTaskFails(t *testing.T) {
	expectedUUID := uuid.NewUUID().String()
	task := mockTask(expectedUUID, nil, errors.New("i fail soz"))

	throttleCalled := make(chan struct{}, 1)
	opened := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)

	throttle := mockThrottle(time.Millisecond*50, throttleCalled)

	iter := mockIterWithCollectionSize(expectedUUID, 2000, closed)
	happyIter(iter)

	tx := mockTx(iter, nil)
	db := mockDB(opened, tx, nil)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("name", uuidCollectionBuilder, "collection", "origin", time.Millisecond*50, throttle, task)

	c.Start()

	<-opened
	<-closed

	<-throttleCalled

	c.Stop()

	<-throttleCalled

	assert.Len(t, c.State(), 1)
	assert.Contains(t, c.State(), stoppedState)

	mock.AssertExpectationsForObjects(t, throttle, iter, tx, db, task)
	assert.Equal(t, 1, c.Metadata().Errors)
}

func TestWholeCollectionCycleRunCompleted(t *testing.T) {
	expectedUUID := uuid.NewUUID().String()
	collectionSize := 10

	task := mockTask(expectedUUID, nil, nil)

	throttleCalled := make(chan struct{}, 1)
	opened := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)

	throttle := mockThrottle(time.Millisecond*50, throttleCalled)

	iter := mockIterWithCollectionSize(expectedUUID, collectionSize, closed)
	happyIter(iter)

	tx := mockTx(iter, nil)
	db := mockDB(opened, tx, nil)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("name", uuidCollectionBuilder, "collection", "origin", time.Millisecond*50, throttle, task)

	c.Start()

	<-opened
	<-closed

	for i := 0; i < collectionSize; i++ {
		<-throttleCalled
	}

	<-opened

	assert.Equal(t, 1, c.Metadata().Iteration)
	assert.Equal(t, collectionSize, c.Metadata().Completed)

	<-closed

	for i := 0; i < 3; i++ {
		<-throttleCalled
	}

	c.Stop()
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, c.State(), 1)
	assert.Contains(t, c.State(), stoppedState)

	mock.AssertExpectationsForObjects(t, throttle, iter, tx, db, task)
	assert.Equal(t, 0, c.Metadata().Errors)
	assert.Equal(t, 2, c.Metadata().Iteration)
	assert.Equal(t, 3, c.Metadata().Completed)
}

func TestWholeCollectionCycleIterationError(t *testing.T) {
	expectedUUID := uuid.NewUUID().String()

	task := new(tasks.MockTask)

	opened := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)

	throttle := new(MockThrottle)

	iter := mockIter(expectedUUID, false, closed)
	iter.On("Err").Return(errors.New("ruh-roh"))

	tx := mockTx(iter, nil)
	db := mockDB(opened, tx, nil)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("name", uuidCollectionBuilder, "collection", "origin", time.Millisecond*50, throttle, task)

	c.Start()

	<-opened
	<-closed

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, c.State(), 2)
	assert.Contains(t, c.State(), stoppedState)
	assert.Contains(t, c.State(), unhealthyState)

	mock.AssertExpectationsForObjects(t, throttle, iter, tx, db, task)
}

func TestWholeCollectionCycleMongoDBConnectionError(t *testing.T) {
	task := new(tasks.MockTask)

	opened := make(chan struct{}, 1)

	throttle := new(MockThrottle)

	tx := new(native.MockTX)
	db := mockDB(opened, tx, errors.New("nein"))

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("name", uuidCollectionBuilder, "collection", "origin", time.Millisecond*50, throttle, task)

	c.Start()
	<-opened

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, c.State(), 2)
	assert.Contains(t, c.State(), stoppedState)
	assert.Contains(t, c.State(), unhealthyState)

	mock.AssertExpectationsForObjects(t, throttle, tx, db, task)
}

func TestWholeCollectionCycleRunEmptyCollection(t *testing.T) {
	opened := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)

	iter := new(native.MockDBIter)
	iter.On("Close").Run(func(arg1 mock.Arguments) {
		closed <- struct{}{}
	}).Return(nil)

	tx := new(native.MockTX)
	tx.On("FindUUIDs", "a-collection", 0, 100).Return(iter, 0, nil)

	db := mockDB(opened, tx, nil)

	task := new(tasks.MockTask)
	throttle := new(MockThrottle)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("test-cycle", uuidCollectionBuilder, "a-collection", "a-origin-id", 1*time.Second, throttle, task)
	c.Start()

	<-opened
	<-closed

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, c.State(), 2)
	assert.Contains(t, c.State(), stoppedState)
	assert.Contains(t, c.State(), unhealthyState)

	mock.AssertExpectationsForObjects(t, db, tx, task, throttle)
}

func TestThrottledWholeCollectionTransformToConfig(t *testing.T) {
	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	iter := new(native.MockDBIter)
	task := new(tasks.MockTask)
	throttle := new(MockThrottle)

	throttle.On("Interval").Return(time.Minute)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("test-cycle", uuidCollectionBuilder, "a-collection", "a-origin-id", 1*time.Second, throttle, task)

	conf := c.TransformToConfig()
	assert.Equal(t, "a-collection", conf.Collection)
	assert.Equal(t, "a-origin-id", conf.Origin)
	assert.Equal(t, "test-cycle", conf.Name)
	assert.Equal(t, "ThrottledWholeCollection", conf.Type)
	assert.Equal(t, time.Second.String(), conf.CoolDown)
	assert.Equal(t, time.Minute.String(), conf.Throttle)

	mock.AssertExpectationsForObjects(t, db, mockTx, task, iter, throttle)
}

type atomicInt struct {
	sync.Mutex
	val int
}

func mockDB(opened chan struct{}, tx native.TX, err error) *native.MockDB {
	db := new(native.MockDB)
	db.On("Open").Return(tx, err).Run(func(arg1 mock.Arguments) {
		opened <- struct{}{}
	})
	return db
}

func mockTx(iter native.DBIter, err error) *native.MockTX {
	mockTx := new(native.MockTX)
	mockTx.On("FindUUIDs", "collection", 0, 100).Return(iter, 15, err)
	return mockTx
}

func mockIter(expectedUUID string, moreItems bool, closed chan struct{}) *native.MockDBIter {
	iter := new(native.MockDBIter)
	iter.On("Next", mock.MatchedBy(func(arg *map[string]interface{}) bool {
		m := *arg
		m["uuid"] = bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(expectedUUID))}
		return true
	})).Return(moreItems)

	iter.On("Close").Run(func(arg1 mock.Arguments) {
		closed <- struct{}{}
	}).Return(nil)

	return iter
}

func mockIterWithCollectionSize(expectedUUID string, collectionSize int, closed chan struct{}) *native.MockDBIter {
	count := &atomicInt{val: 1}
	iter := new(native.MockDBIter)

	next := func(args map[string]interface{}) {
		count.val++
		args["uuid"] = bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(expectedUUID))}
	}

	iter.On("Next", mock.MatchedBy(func(arg *map[string]interface{}) bool {
		count.Lock()
		defer count.Unlock()

		if count.val%collectionSize == 0 {
			return false
		}

		next(*arg)
		return true
	})).Return(true) // if this isn't the end of the collection, return true to continue the iteration

	iter.On("Next", mock.MatchedBy(func(arg *map[string]interface{}) bool {
		count.Lock()
		defer count.Unlock()

		if count.val%collectionSize != 0 {
			return false
		}

		next(*arg)
		return true
	})).Return(false) // if this is the end of the collection return false to finish the iteration

	iter.On("Close").Run(func(arg1 mock.Arguments) {
		closed <- struct{}{}
	}).Return(nil)

	return iter
}

func happyIter(iter *native.MockDBIter) {
	iter.On("Err").Return(nil)
	iter.On("Timeout").Return(false)
}

func mockThrottle(interval time.Duration, called chan struct{}) *MockThrottle {
	throttle := new(MockThrottle)
	throttle.On("Queue").Run(func(arg1 mock.Arguments) {
		time.Sleep(interval)
		called <- struct{}{}
	}).Return(nil)
	return throttle
}

func mockTask(expectedUUID string, prepErr error, execErr error) *tasks.MockTask {
	task := new(tasks.MockTask)

	task.On("Prepare", "collection", expectedUUID).Return(&native.Content{}, "tid_"+expectedUUID, prepErr)
	if prepErr != nil {
		return task
	}

	task.On("Execute", expectedUUID, mock.AnythingOfType("*native.Content"), "origin", "tid_"+expectedUUID).Return(execErr)
	return task
}

func TestWholeCollectionCycleCanBeStoppedEvenIfNotStarted(t *testing.T) {
	db := new(native.MockDB)
	task := new(tasks.MockTask)
	throttle := new(MockThrottle)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(db, nil, blacklist.NoOpBlacklist)

	c := NewThrottledWholeCollectionCycle("test-cycle", uuidCollectionBuilder, "a-collection", "a-origin-id", 1*time.Second, throttle, task)
	c.Stop()

	mock.AssertExpectationsForObjects(t, db, task, throttle)
}
