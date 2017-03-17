package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type FixedWindowCycle struct {
	*abstractCycle
	timeWindow      time.Duration
	minimumThrottle time.Duration
	TimeWindow      string `json:"timeWindow"`
	MinimumThrottle string `json:"minimumThrottle"`
}

func NewFixedWindowCycle(name string, db native.DB, dbCollection string, origin string, coolDown time.Duration, timeWindow time.Duration, minimumThrottle time.Duration, publishTask tasks.Task) Cycle {
	return &FixedWindowCycle{
		newAbstractCycle(name, "FixedWindow", db, dbCollection, origin, coolDown, publishTask),
		timeWindow,
		minimumThrottle,
		timeWindow.String(),
		minimumThrottle.String(),
	}
}

func (s *FixedWindowCycle) Start() {
	log.WithField("collection", s.DBCollection).WithField("name", s.Name).WithField("timeWindow", s.timeWindow).Info("Starting fixed window cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.Metadata().UpdateState(startingState)
	go s.start(ctx)
}

func (s *FixedWindowCycle) start(ctx context.Context) {
	startTime := time.Now().Add(-1 * s.timeWindow)
	for {
		endTime := startTime.Add(s.timeWindow)
		uuidCollection, err := native.NewNativeUUIDCollectionForTimeWindow(s.db, s.DBCollection, startTime, endTime)
		if err != nil {
			log.WithError(err).WithField("start", startTime).WithField("end", endTime).Warn("Failed to query native collection for time window.")
			s.Metadata().UpdateState(coolDownState)
			time.Sleep(s.coolDown)
			continue
		}

		copiedStartTime := startTime // Copy so that we don't change the time for the cycle
		s.CycleMetadata = &CycleMetadata{State: runningState, Iteration: s.CycleMetadata.Iteration + 1, Total: uuidCollection.Length(), Start: &copiedStartTime, End: &endTime, lock: &sync.RWMutex{}}
		startTime = endTime

		if uuidCollection.Length() == 0 {
			s.CycleMetadata.UpdateState(coolDownState)
			time.Sleep(s.timeWindow)
			continue
		}

		t, cancel := NewDynamicThrottle(s.minimumThrottle, s.timeWindow, uuidCollection.Length()+1, 1) // add one to the length to increase the wait time
		stopped, err := s.publishCollection(ctx, uuidCollection, t)
		if stopped {
			break
		}

		if err != nil {
			log.WithError(err).WithField("collection", s.DBCollection).WithField("id", s.ID).Error("Unexpected error occurred while publishing collection.")
			break
		}

		t.Queue() // ensure we wait a reasonable amount of time before the next iteration
		cancel()
	}

	s.Metadata().UpdateState(stoppedState)
}

func (s *FixedWindowCycle) UpdateConfiguration() {

}

func (s *FixedWindowCycle) TransformToConfig() *CycleConfig {
	return &CycleConfig{Name: s.Name, Type: s.Type, Collection: s.DBCollection, TimeWindow: s.TimeWindow}
}
