package scheduler

import (
	"errors"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWholeCollectionCycleRunWithMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
	}

	iter := new(native.MockDBIter)
	iter.On("Next", mock.MatchedBy(func(arg *map[string]interface{}) bool {
		m := *arg
		m["uuid"] = bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse("583c9c34-8d28-486d-92a0-0e53ff7744d7"))}
		return true
	})).Return(true)

	iter.On("Close").Return(nil)
	iter.On("Timeout").Return(false)

	mockTx := new(native.MockTX)
	mockTx.On("FindUUIDs", "a-collection", 500, 80).Return(iter, 12, nil)

	db := new(native.MockDB)
	db.On("Open").Return(mockTx, nil).After(1 * time.Second)

	task := new(tasks.MockTask)
	task.On("Publish", "a-origin-id", "a-collection", "583c9c34-8d28-486d-92a0-0e53ff7744d7").Return(nil)

	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)
	throttle.On("Queue").Run(func(arg1 mock.Arguments) {
		time.Sleep(250 * time.Millisecond)
	}).Return(nil)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)

	metadata := CycleMetadata{Completed: 500, Iteration: 1}
	c.SetMetadata(metadata)

	c.Start()

	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")

	time.Sleep(2 * time.Second)
	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), runningState, "Cycle should be in running state")

	c.Stop()
	time.Sleep(1 * time.Second)

	assert.Contains(t, c.State(), stoppedState, "Cycle should be in stopped state")

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	throttle.AssertExpectations(t)
	iter.AssertExpectations(t)

	assert.Equal(t, 1, c.Metadata().Iteration)
	assert.True(t, c.Metadata().Completed >= 501)
}

func TestWholeCollectionCycleTaskFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
	}

	iter := new(native.MockDBIter)
	iter.On("Next", mock.MatchedBy(func(arg *map[string]interface{}) bool {
		m := *arg
		m["uuid"] = bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse("583c9c34-8d28-486d-92a0-0e53ff7744d7"))}
		return true
	})).Return(true)

	iter.On("Close").Return(nil)
	iter.On("Timeout").Return(false)

	mockTx := new(native.MockTX)
	mockTx.On("FindUUIDs", "a-collection", 500, 80).Return(iter, 12, nil)

	db := new(native.MockDB)
	db.On("Open").Return(mockTx, nil).After(1 * time.Second)

	task := new(tasks.MockTask)
	task.On("Publish", "a-origin-id", "a-collection", "583c9c34-8d28-486d-92a0-0e53ff7744d7").Return(errors.New("i fail soz"))

	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)
	throttle.On("Queue").Run(func(arg1 mock.Arguments) {
		time.Sleep(250 * time.Millisecond)
	}).Return(nil)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)

	metadata := CycleMetadata{Completed: 500, Iteration: 1}
	c.SetMetadata(metadata)

	c.Start()

	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")

	time.Sleep(2 * time.Second)
	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), runningState, "Cycle should be in running state")

	c.Stop()
	time.Sleep(1 * time.Second)

	assert.Contains(t, c.State(), stoppedState, "Cycle should be in stopped state")

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	throttle.AssertExpectations(t)
	iter.AssertExpectations(t)

	assert.Equal(t, 1, c.Metadata().Iteration)
	assert.True(t, c.Metadata().Completed >= 501)
	assert.True(t, c.Metadata().Errors >= 1)
}

func TestWholeCollectionCycleRunCompleted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
	}

	iter := new(native.MockDBIter)
	iter.On("Next", mock.Anything).Return(false)
	iter.On("Close").Return(nil)
	iter.On("Err").Return(nil)
	iter.On("Timeout").Return(false)

	mockTx := new(native.MockTX)
	mockTx.On("FindUUIDs", "a-collection", 0, 80).Return(iter, 12, nil)

	db := new(native.MockDB)
	db.On("Open").Return(mockTx, nil).After(1 * time.Second)

	task := new(tasks.MockTask)

	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)
	throttle.On("Queue").Run(func(arg1 mock.Arguments) {
		time.Sleep(250 * time.Millisecond)
	}).Return(nil)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)

	metadata := CycleMetadata{Completed: 0, Iteration: 1}
	c.SetMetadata(metadata)

	c.Start()

	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")

	time.Sleep(2 * time.Second)
	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), runningState, "Cycle should be in running state")

	c.Stop()
	time.Sleep(1 * time.Second)

	assert.Contains(t, c.State(), stoppedState, "Cycle should be in stopped state")

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	throttle.AssertExpectations(t)
	iter.AssertExpectations(t)

	assert.Equal(t, 2, c.Metadata().Iteration)
}

func TestWholeCollectionCycleError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
	}

	iter := new(native.MockDBIter)
	iter.On("Next", mock.Anything).Return(false)
	iter.On("Close").Return(nil)
	iter.On("Err").Return(errors.New("ruh-roh"))

	mockTx := new(native.MockTX)
	mockTx.On("FindUUIDs", "a-collection", 0, 80).Return(iter, 12, nil)

	db := new(native.MockDB)
	db.On("Open").Return(mockTx, nil).After(1 * time.Second)

	task := new(tasks.MockTask)

	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)
	throttle.On("Queue").Run(func(arg1 mock.Arguments) {
		time.Sleep(250 * time.Millisecond)
	}).Return(nil)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)

	c.Start()

	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")

	time.Sleep(2 * time.Second)

	assert.Contains(t, c.State(), stoppedState)
	assert.Contains(t, c.State(), unhealthyState)

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	throttle.AssertExpectations(t)
	iter.AssertExpectations(t)
}

func TestWholeCollectionCycleRunEmptyUUID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
	}

	iter := new(native.MockDBIter)
	iter.On("Next", mock.Anything).Return(true)
	iter.On("Close").Return(nil)
	iter.On("Timeout").Return(false)

	mockTx := new(native.MockTX)
	mockTx.On("FindUUIDs", "a-collection", 0, 80).Return(iter, 12, nil)

	db := new(native.MockDB)
	db.On("Open").Return(mockTx, nil).After(1 * time.Second)

	task := new(tasks.MockTask)

	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)
	throttle.On("Queue").Run(func(arg1 mock.Arguments) {
		time.Sleep(250 * time.Millisecond)
	}).Return(nil)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)

	c.Start()

	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")

	time.Sleep(2 * time.Second)
	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), runningState, "Cycle should be in running state")

	c.Stop()
	time.Sleep(1 * time.Second)

	assert.Contains(t, c.State(), stoppedState, "Cycle should be in stopped state")

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	throttle.AssertExpectations(t)
	iter.AssertExpectations(t)
}

func TestWholeCollectionCycleRunMongoDBConnectionError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
	}

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, errors.New("error in DB connection")).After(1 * time.Second)

	task := new(tasks.MockTask)
	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)
	c.Start()

	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")

	time.Sleep(2 * time.Second)
	assert.Len(t, c.State(), 2, "The cycle should have one state")
	assert.Contains(t, c.State(), stoppedState, "Cycle should be in stopped state")
	assert.Contains(t, c.State(), unhealthyState, "Cycle should be in unhealthy state")

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	throttle.AssertExpectations(t)
}

func TestWholeCollectionCycleRunEmptyCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
	}

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	iter := new(native.MockDBIter)
	db.On("Open").Return(mockTx, nil).After(1 * time.Second)
	mockTx.On("FindUUIDs", "a-collection", 0, 80).Return(iter, 0, nil)
	iter.On("Close").Return(nil)

	task := new(tasks.MockTask)
	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)
	c.Start()

	assert.Len(t, c.State(), 1, "The cycle should have one state")
	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")

	time.Sleep(2 * time.Second)
	assert.Len(t, c.State(), 2, "The cycle should have one state")
	assert.Contains(t, c.State(), stoppedState, "Cycle should be in stopped state")
	assert.Contains(t, c.State(), unhealthyState, "Cycle should be in unhealthy state")

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	throttle.AssertExpectations(t)
}

func TestThrottledWholeCollectionTransformToConfig(t *testing.T) {
	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	iter := new(native.MockDBIter)
	task := new(tasks.MockTask)
	throttle := new(MockThrottle)

	throttle.On("Interval").Return(time.Minute)

	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)

	conf := c.TransformToConfig()
	assert.Equal(t, "a-collection", conf.Collection)
	assert.Equal(t, "a-origin-id", conf.Origin)
	assert.Equal(t, "test-cycle", conf.Name)
	assert.Equal(t, "ThrottledWholeCollection", conf.Type)
	assert.Equal(t, time.Second.String(), conf.CoolDown)
	assert.Equal(t, time.Minute.String(), conf.Throttle)

	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	task.AssertExpectations(t)
	iter.AssertExpectations(t)
	throttle.AssertExpectations(t)
}
