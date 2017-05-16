package scheduler

import (
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/blacklist"
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
	s := NewScheduler(blacklist.NoOpBlacklist, db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")
	c1.On("Start").Return()
	c2.On("Start").Return()
	c1.On("TransformToConfig").Return(&CycleConfig{Type: "test"})
	c2.On("TransformToConfig").Return(&CycleConfig{Type: "test"})

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
	s := NewScheduler(blacklist.NoOpBlacklist, db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

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
	s := NewScheduler(blacklist.NoOpBlacklist, db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")
	c1.On("Start").Return()
	c2.On("Start").Return()
	c1.On("Stop").Return()
	c2.On("Stop").Return()
	c1.On("TransformToConfig").Return(&CycleConfig{Type: "test"})
	c2.On("TransformToConfig").Return(&CycleConfig{Type: "test"})

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
	s := NewScheduler(blacklist.NoOpBlacklist, db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute)

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
	c1.On("TransformToConfig").Return(&CycleConfig{Type: "test"})

	db := new(native.MockDB)
	dbCollection := "testCollection"
	origin := "testOrigin"
	coolDown := time.Minute
	throttle, _ := NewThrottle(time.Second, 1)
	c2 := NewThrottledWholeCollectionCycle("test", blacklist.NoOpBlacklist, db, dbCollection, origin, coolDown, throttle, nil)
	id2 := c2.ID()

	rw := MockMetadataRW{}
	rw.On("WriteMetadata", id2, c2).Return(nil)

	s := NewScheduler(blacklist.NoOpBlacklist, db, &tasks.MockTask{}, &rw, 1*time.Minute)

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.SaveCycleMetadata()

	rw.AssertExpectations(t)
}

func TestCalculateArchiveCycleStartInterval(t *testing.T) {
	assert := assert.New(t)
	id1 := "id1"

	c1 := new(MockCycle)
	c1.On("ID").Return(id1)
	c1.On("Start").Return()
	c1.On("TransformToConfig").Return(&CycleConfig{Type: "test"})

	db := new(native.MockDB)
	dbCollection := "testCollection"
	origin := "testOrigin"
	coolDown := time.Minute
	throttle, _ := NewThrottle(time.Second, 1)
	throttle2, _ := NewThrottle(time.Second, 2)
	c2 := NewThrottledWholeCollectionCycle("test", blacklist.NoOpBlacklist, db, dbCollection, origin, coolDown, throttle, nil)
	c3 := NewThrottledWholeCollectionCycle("test2", blacklist.NoOpBlacklist, db, dbCollection, origin, coolDown, throttle2, nil)

	rw := MockMetadataRW{}

	s := NewScheduler(blacklist.NoOpBlacklist, db, &tasks.MockTask{}, &rw, 1*time.Minute)

	s.AddCycle(c1)
	s.AddCycle(c2)
	s.AddCycle(c3)

	testIterval := s.(*defaultScheduler).archiveCycleStartInterval()

	//we have 2 archive cycles out of 3, cycles and the shortest throttle is 1 sec
	//therefore the startup interval is 500ms
	expected, _ := time.ParseDuration("500ms")
	assert.Equal(expected, testIterval, "test interval should be 500ms")
}
