package scheduler

import (
	"context"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	log "github.com/Sirupsen/logrus"
)

type abstractTimeWindowedCycle struct {
	*abstractCycle
	timeWindow      time.Duration
	minimumThrottle time.Duration
	batchDuration   time.Duration

	TimeWindow      string `json:"timeWindow"`
	MinimumThrottle string `json:"minimumThrottle"`
}

func newAbstractTimeWindowedCycle(base *abstractCycle, timeWindow time.Duration, minimumThrottle time.Duration, batchDuration time.Duration) *abstractTimeWindowedCycle {
	return &abstractTimeWindowedCycle{
		base,
		timeWindow,
		minimumThrottle,
		batchDuration,
		timeWindow.String(),
		minimumThrottle.String(),
	}
}

func (s *abstractTimeWindowedCycle) start(ctx context.Context, throttle func(publishes int) (Throttle, context.CancelFunc)) {
	endTime := time.Now()
	startTime := endTime.Add(-1 * s.timeWindow)

	b := true
	for b {
		endTime, b = s.publishCollectionCycle(ctx, startTime, endTime, throttle)
	}
}

func (s *abstractTimeWindowedCycle) publishCollectionCycle(ctx context.Context, startTime time.Time, endTime time.Time, throttle func(publishes int) (Throttle, context.CancelFunc)) (time.Time, bool) {
	uuidCollection, err := native.NewNativeUUIDCollectionForTimeWindow(s.db, s.DBCollection, startTime, endTime, s.batchDuration)
	if err != nil {
		log.WithField("id", s.CycleID).WithField("name", s.Name).WithField("collection", s.DBCollection).WithField("start", startTime).WithField("end", endTime).WithError(err).Warn("Failed to query native collection for time window.")
		s.UpdateState(stoppedState, unhealthyState)
		return endTime, false
	}
	defer uuidCollection.Close()

	copiedTime := startTime // Copy so that we don't change the time for the cycle

	metadata := CycleMetadata{State: []string{runningState}, Attempts: s.CycleMetadata.Attempts + 1, Total: uuidCollection.Length(), Start: &copiedTime, End: &endTime, state: make(map[string]struct{})}
	s.SetMetadata(metadata)

	startTime = endTime

	if uuidCollection.Length() == 0 {
		endTime = s.performCooldown(coolDownState)
		return endTime, true
	}

	t, cancel := throttle(uuidCollection.Length() + 1) // add one to the length to increase the wait time
	stopped, err := s.publishCollection(ctx, uuidCollection, t)

	cancel()
	if stopped {
		s.UpdateState(stoppedState)
		return endTime, false
	}

	if err != nil {
		log.WithField("id", s.CycleID).WithField("name", s.Name).WithField("collection", s.DBCollection).WithError(err).Warn("Unexpected error occurred while publishing collection.")
		s.UpdateState(stoppedState, unhealthyState)
		return endTime, false
	}

	t.Queue() // ensure we wait a reasonable amount of time before the next iteration
	return time.Now(), true
}

func (s *abstractTimeWindowedCycle) performCooldown(states ...string) time.Time {
	s.UpdateState(states...)
	time.Sleep(s.coolDown)
	return time.Now()
}
