package scheduler

import (
	"errors"
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/stretchr/testify/assert"
)

// func TestWholeCollectionCycleRun(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping - this test can take several seconds.")
// 		return
// 	}
//
// 	db := new(native.MockDB)
// 	mockTx := new(native.MockTX)
// 	iter := new(native.MockDBIter)
// 	db.On("Open").Return(mockTx, nil).After(1 * time.Second)
// 	mockTx.On("FindUUIDs", "a-collection", 0, 80).Return(iter, 12, nil)
// 	iter.On("Next", mock.Anything).Return(true)
// 	iter.On("Close").Return(nil)
// 	iter.On("Timeout").Return(false)
//
// 	task := new(tasks.MockTask)
// 	throttle := new(MockThrottle)
// 	throttle.On("Interval").Return(1 * time.Second)
// 	throttle.On("Queue").Return(nil)
//
// 	c := NewThrottledWholeCollectionCycle("test-cycle", db, "a-collection", "a-origin-id", 1*time.Second, throttle, task)
// 	c.Start()
//
// 	assert.Len(t, c.State(), 1, "The cycle should have one state")
// 	assert.Contains(t, c.State(), startingState, "Cycle should be in running state")
//
// 	time.Sleep(2 * time.Second)
// 	assert.Len(t, c.State(), 1, "The cycle should have one state")
// 	assert.Contains(t, c.State(), runningState, "Cycle should be in running state")
//
// 	c.Stop()
// 	time.Sleep(1 * time.Second)
//
// 	assert.Contains(t, c.State(), stoppedState, "Cycle should be in stopped state")
//
// 	db.AssertExpectations(t)
// 	mockTx.AssertExpectations(t)
// 	task.AssertExpectations(t)
// 	throttle.AssertExpectations(t)
// 	iter.AssertExpectations(t)
// }

func TestWholeCollectionCycleRunMongoDBConnectionError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
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
		return
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
