package scheduler

import (
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/stretchr/testify/assert"
)

const cycleConfigFile = "test/cycle_test.yml"
const expectedColletion = "a-collection"

func TestSchedulerShouldStartWhenEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}

	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")
	c1.On("Start").Return()
	c2.On("Start").Return()

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.ToggleHandler("true")
	err := s.Start()
	assert.NoError(t, err, "It should not return an error to Start")

	time.Sleep(2 * time.Second) // wait cycles to start

	db.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
}

func TestSchedulerDoNotStartWhenDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}

	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.ToggleHandler("false")
	err := s.Start()
	assert.EqualError(t, err, "Scheduler is not enabled", "It should not return an error to Start")

	time.Sleep(2 * time.Second) // wait cycles to start

	db.AssertExpectations(t)
	c1.AssertNotCalled(t, "Start")
	c2.AssertNotCalled(t, "Start")
}

func TestSchedulerResumeAfterDisable(t *testing.T) {
	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")
	c1.On("Start").Return()
	c2.On("Start").Return()
	c1.On("Stop").Return()
	c2.On("Stop").Return()

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.ToggleHandler("true")
	err := s.Start()
	assert.NoError(t, err, "It should not return an error to Start")

	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNotCalled(t, "Stop")
	c2.AssertNotCalled(t, "Stop")

	s.ToggleHandler("false")

	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)

	s.ToggleHandler("true")

	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)

	s.Start()

	c1.AssertNumberOfCalls(t, "Start", 2)
	c2.AssertNumberOfCalls(t, "Start", 2)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)

	s.Shutdown()

	c1.AssertNumberOfCalls(t, "Start", 2)
	c2.AssertNumberOfCalls(t, "Start", 2)
	c1.AssertNumberOfCalls(t, "Stop", 2)
	c2.AssertNumberOfCalls(t, "Stop", 2)

	s.ToggleHandler("true")

	c1.AssertNumberOfCalls(t, "Start", 2)
	c2.AssertNumberOfCalls(t, "Start", 2)
	c1.AssertNumberOfCalls(t, "Stop", 2)
	c2.AssertNumberOfCalls(t, "Stop", 2)

	db.AssertExpectations(t)
}

func TestSchedulerInvalidToggleValue(t *testing.T) {
	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.ToggleHandler("invalid-value")
	err := s.Start()
	assert.EqualError(t, err, "Scheduler is not enabled", "It should return an error to Start")
	c1.AssertNotCalled(t, "Start")
	c2.AssertNotCalled(t, "Start")
}

func TestSaveCycleMetadata(t *testing.T) {
	id1 := "id1"

	c1 := new(MockCycle)
	c1.On("ID").Return(id1)
	c1.On("Start").Return()

	db := new(native.MockDB)
	dbCollection := "testCollection"
	origin := "testOrigin"
	coolDown := time.Minute
	throttle, _ := NewThrottle(time.Second, 1)
	c2 := NewThrottledWholeCollectionCycle("test", db, dbCollection, origin, coolDown, throttle, nil)
	id2 := c2.ID()

	rw := MockMetadataRW{}
	rw.On("WriteMetadata", id2, c2).Return(nil)

	s := NewScheduler(db, &tasks.MockTask{}, &rw, 1*time.Minute)

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.SaveCycleMetadata()

	rw.AssertExpectations(t)
}
