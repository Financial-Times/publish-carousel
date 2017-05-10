package scheduler

import (
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const cycleConfigFile = "test/cycle_test.yml"
const expectedColletion = "a-collection"

func TestSchedulerShouldStartWhenEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}

	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")
	c1.On("Start").Return()
	c2.On("Start").Return()

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.ManualToggleHandler("true")
	s.AutomaticToggleHandler("true")
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
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.ManualToggleHandler("false")
	s.AutomaticToggleHandler("true")
	err := s.Start()
	assert.EqualError(t, err, "Scheduler is not enabled", "It should not return an error to Start")

	time.Sleep(2 * time.Second) // wait cycles to start

	db.AssertExpectations(t)
	c1.AssertNotCalled(t, "Start")
	c2.AssertNotCalled(t, "Start")
}

func TestSchedulerResumeAfterDisable(t *testing.T) {
	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

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

	s.ManualToggleHandler("true")
	s.AutomaticToggleHandler("true")
	err := s.Start()
	assert.NoError(t, err, "It should not return an error to Start")

	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNotCalled(t, "Stop")
	c2.AssertNotCalled(t, "Stop")

	s.ManualToggleHandler("false")

	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)

	s.ManualToggleHandler("true")

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

	s.ManualToggleHandler("true")

	c1.AssertNumberOfCalls(t, "Start", 2)
	c2.AssertNumberOfCalls(t, "Start", 2)
	c1.AssertNumberOfCalls(t, "Stop", 2)
	c2.AssertNumberOfCalls(t, "Stop", 2)

	db.AssertExpectations(t)
}

func TestSchedulerInvalidToggleValue(t *testing.T) {
	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)

	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.ManualToggleHandler("invalid-value")
	err := s.Start()
	assert.EqualError(t, err, "Scheduler is not enabled", "It should return an error to Start")
	c1.AssertNotCalled(t, "Start")
	c2.AssertNotCalled(t, "Start")
}

func TestAutomaticToggleDisabledAndManualToggleEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}

	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

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
	s.ManualToggleHandler("true")
	s.AutomaticToggleHandler("true")
	err := s.Start()
	assert.NoError(t, err, "It should not return an error to Start")

	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(1 * time.Second)
			s.ManualToggleHandler("true")
		}
	}()
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(1 * time.Second)
			if i%2 == 1 {
				s.AutomaticToggleHandler("false")
			} else {
				s.AutomaticToggleHandler("true")
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	assert.True(t, s.IsRunning(), "The scheduler should be in running state")
	assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should not be in autoDisabled state")
	assert.True(t, s.IsEnabled(), "The scheduler should be in enabled state")
	assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")

	time.Sleep(2 * time.Second)
	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)
	assert.False(t, s.IsRunning(), "The scheduler should not be in running state")
	assert.True(t, s.IsAutomaticallyDisabled(), "The scheduler should be in autoDisabled state")
	assert.False(t, s.IsEnabled(), "The scheduler should not be in enabled state")
	assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")

	time.Sleep(1 * time.Second)
	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)
	assert.False(t, s.IsRunning(), "The scheduler should not be in running state")
	assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should be in autoDisabled state")
	assert.True(t, s.IsEnabled(), "The scheduler should be in enabled state")
	assert.True(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should be autoDisabled state")

	s.Start()
	time.Sleep(250 * time.Millisecond)
	c1.AssertNumberOfCalls(t, "Start", 2)
	c2.AssertNumberOfCalls(t, "Start", 2)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)
	assert.True(t, s.IsRunning(), "The scheduler should be in running state")
	assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should not be in autoDisabled state")
	assert.True(t, s.IsEnabled(), "The scheduler should be in enabled state")
	assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")

	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
}

func TestAutomaticToggleEnabledAndManualToggleDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}
	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

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
	s.ManualToggleHandler("true")
	s.AutomaticToggleHandler("true")
	err := s.Start()
	assert.NoError(t, err, "It should not return an error to Start")

	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(1 * time.Second)
			s.AutomaticToggleHandler("true")
		}
	}()
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(1 * time.Second)
			if i%2 == 1 {
				s.ManualToggleHandler("false")
			} else {
				s.ManualToggleHandler("true")
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	assert.True(t, s.IsRunning(), "The scheduler should be in running state")
	assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should not be in autoDisabled state")
	assert.True(t, s.IsEnabled(), "The scheduler should be in enabled state")
	assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")

	time.Sleep(2 * time.Second)
	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)
	assert.False(t, s.IsRunning(), "The scheduler should not be in running state")
	assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should not be in autoDisabled state")
	assert.False(t, s.IsEnabled(), "The scheduler should not be in enabled state")
	assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")

	time.Sleep(1 * time.Second)
	c1.AssertNumberOfCalls(t, "Start", 1)
	c2.AssertNumberOfCalls(t, "Start", 1)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)
	assert.False(t, s.IsRunning(), "The scheduler should not be in running state")
	assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should be in autoDisabled state")
	assert.True(t, s.IsEnabled(), "The scheduler should be in enabled state")
	assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")

	s.Start()
	time.Sleep(250 * time.Millisecond)
	c1.AssertNumberOfCalls(t, "Start", 2)
	c2.AssertNumberOfCalls(t, "Start", 2)
	c1.AssertNumberOfCalls(t, "Stop", 1)
	c2.AssertNumberOfCalls(t, "Stop", 1)
	assert.True(t, s.IsRunning(), "The scheduler should be in running state")
	assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should not be in autoDisabled state")
	assert.True(t, s.IsEnabled(), "The scheduler should be in enabled state")
	assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")

	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
}

func TestAutomaticToggleFlappingAndManualToggleDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}
	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)
	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")

	s.AddCycle(c1)
	s.AddCycle(c2)
	s.ManualToggleHandler("false")
	s.AutomaticToggleHandler("true")
	err := s.Start()
	assert.Error(t, err, "It should return an error to Start")

	go func() {
		for i := 0; i < 4; i++ {
			time.Sleep(1 * time.Second)
			if i%2 == 1 {
				s.AutomaticToggleHandler("false")
			} else {
				s.AutomaticToggleHandler("true")
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 4; i++ {
		assert.False(t, s.IsRunning(), "The scheduler should not be in running state")
		assert.False(t, s.IsAutomaticallyDisabled(), "The scheduler should not be in autoDisabled state")
		assert.False(t, s.IsEnabled(), "The scheduler should not be in enabled state")
		assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")
		time.Sleep(1 * time.Second)
	}
}

func TestAutomaticToggleDisabledAndManualToggleFlapping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}
	db := new(native.MockDB)
	s := NewScheduler(db, &tasks.MockTask{}, &MockMetadataRW{}, 1*time.Minute, 1*time.Minute)

	c1 := new(MockCycle)
	c2 := new(MockCycle)
	c1.On("ID").Return("id1")
	c2.On("ID").Return("id2")

	s.AddCycle(c1)
	s.AddCycle(c2)
	s.ManualToggleHandler("true")
	s.AutomaticToggleHandler("false")
	err := s.Start()
	assert.Error(t, err, "It should return an error to Start")

	go func() {
		for i := 0; i < 4; i++ {
			time.Sleep(1 * time.Second)
			if i%2 == 1 {
				s.ManualToggleHandler("false")
			} else {
				s.ManualToggleHandler("true")
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 4; i++ {
		assert.False(t, s.IsRunning(), "The scheduler should not be in running state")
		assert.True(t, s.IsAutomaticallyDisabled(), "The scheduler should be in autoDisabled state")
		assert.False(t, s.IsEnabled(), "The scheduler should not be in enabled state")
		assert.False(t, s.WasAutomaticallyDisabled(), "The scheduler's previous state should not be autoDisabled state")
		time.Sleep(1 * time.Second)
	}
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
	rw.On("WriteMetadata", id2, c2.TransformToConfig(), c2.Metadata()).Return(nil)

	s := NewScheduler(db, &tasks.MockTask{}, &rw, 1*time.Minute, 1*time.Minute)

	s.AddCycle(c1)
	s.AddCycle(c2)

	s.(*defaultScheduler).saveCycleMetadata()

	rw.AssertExpectations(t)
}

func TestCheckpointSaveCycleMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}

	dbCollection := "testCollection"
	origin := "testOrigin"
	coolDown := time.Minute

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	iter := new(native.MockDBIter)
	db.On("Open").Return(mockTx, nil).After(1 * time.Second)
	mockTx.On("FindUUIDs", "testCollection", 0, 80).Return(iter, 12, nil)
	iter.On("Next", mock.Anything).Return(true)
	iter.On("Close").Return(nil)
	iter.On("Timeout").Return(false)

	throttle := new(MockThrottle)
	throttle.On("Interval").Return(1 * time.Second)
	throttle.On("Queue").Return(nil).After(1 * time.Second)

	id1 := "id1"
	c1 := new(MockCycle)
	c1.On("ID").Return(id1)
	c1.On("Start").Return()
	c1.On("Stop").Return()

	c2 := NewThrottledWholeCollectionCycle("test", db, dbCollection, origin, coolDown, throttle, nil)
	id2 := c2.ID()

	rw := MockMetadataRW{}
	rw.On("WriteMetadata", id2, c2.TransformToConfig(), mock.AnythingOfType("CycleMetadata")).Return(nil).Times(12)

	s := NewScheduler(db, &tasks.MockTask{}, &rw, 1*time.Second, 1*time.Second)

	s.AddCycle(c1)
	s.AddCycle(c2)
	s.AutomaticToggleHandler("true")
	s.ManualToggleHandler("true")
	err := s.Start()
	assert.NoError(t, err, "It should not return an error to Start")

	time.Sleep(5100 * time.Millisecond)
	rw.AssertNumberOfCalls(t, "WriteMetadata", 5)
	err = s.Shutdown()
	assert.NoError(t, err, "It should not return an error to Shutdown")
	rw.AssertNumberOfCalls(t, "WriteMetadata", 6)

	time.Sleep(2000 * time.Millisecond)
	rw.AssertNumberOfCalls(t, "WriteMetadata", 6)
	err = s.Start()
	assert.NoError(t, err, "It should not return an error to Start")

	time.Sleep(5100 * time.Millisecond)
	rw.AssertNumberOfCalls(t, "WriteMetadata", 11)
	err = s.Shutdown()
	assert.NoError(t, err, "It should not return an error to Shutdown")

	rw.AssertExpectations(t)
}
