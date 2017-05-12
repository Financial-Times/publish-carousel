package scheduler

import (
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckpointSaveCycleMetadata(t *testing.T) {
	// if testing.Short() {
	// 	t.Skip("Skipping - this test can take several seconds.")
	// 	return
	// }

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
	c1.On("TransformToConfig").Return(CycleConfig{Type: "test"})

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
