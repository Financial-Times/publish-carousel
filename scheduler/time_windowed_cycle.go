package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	log "github.com/Sirupsen/logrus"
)

type abstractTimeWindowedCycle struct {
	*abstractCycle
	timeWindow      time.Duration
	minimumThrottle time.Duration

	TimeWindow      string `json:"timeWindow"`
	MinimumThrottle string `json:"minimumThrottle"`
}

func newAbstractTimeWindowedCycle(base *abstractCycle, timeWindow time.Duration, minimumThrottle time.Duration) *abstractTimeWindowedCycle {
	return &abstractTimeWindowedCycle{
		base,
		timeWindow,
		minimumThrottle,
		timeWindow.String(),
		minimumThrottle.String(),
	}
}

func (s *abstractTimeWindowedCycle) start(ctx context.Context, throttle func(publishes int) (Throttle, context.CancelFunc)) {
	endTime := time.Now()
	startTime := endTime.Add(-1 * s.timeWindow)

	for {
		uuidCollection, err := native.NewNativeUUIDCollectionForTimeWindow(s.db, s.DBCollection, startTime, endTime)
		if err != nil {
			log.WithError(err).WithField("start", startTime).WithField("end", endTime).Warn("Failed to query native collection for time window.")
			s.Metadata().UpdateState(unhealthyState, coolDownState)
			time.Sleep(s.coolDown)
			endTime = time.Now()
			continue
		}

		copiedTime := startTime // Copy so that we don't change the time for the cycle
		s.CycleMetadata = &CycleMetadata{State: []string{runningState}, Iteration: s.CycleMetadata.Iteration + 1, Total: uuidCollection.Length(), Start: &copiedTime, End: &endTime, lock: &sync.RWMutex{}, state: make(map[string]struct{})}
		startTime = endTime

		if uuidCollection.Length() == 0 {
			s.Metadata().UpdateState(coolDownState)
			time.Sleep(s.coolDown)
			endTime = time.Now()
			continue
		}

		t, cancel := throttle(uuidCollection.Length() + 1) // add one to the length to increase the wait time
		stopped, err := s.publishCollection(ctx, uuidCollection, t)
		if stopped {
			break
		}

		if err != nil {
			log.WithError(err).WithField("collection", s.DBCollection).WithField("id", s.ID).Error("Unexpected error occurred while publishing collection.")
			s.Metadata().UpdateState(unhealthyState)
			break
		}

		t.Queue() // ensure we wait a reasonable amount of time before the next iteration
		cancel()

		endTime = time.Now()
	}

	s.Metadata().UpdateState(stoppedState)
}
