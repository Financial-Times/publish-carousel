package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type ScalingWindowCycle struct {
	*abstractCycle
	timeWindow      time.Duration
	minimumThrottle time.Duration
	maximumThrottle time.Duration
	TimeWindow      string `json:"timeWindow"`
	MinimumThrottle string `json:"minimumThrottle"`
	MaximumThrottle string `json:"maximumThrottle"`
}

func NewScalingWindowCycle(
	name string,
	db native.DB,
	dbCollection string,
	origin string,
	timeWindow time.Duration,
	coolDown time.Duration,
	minimumThrottle time.Duration,
	maximumThrottle time.Duration,
	publishTask tasks.Task,
) Cycle {

	return &ScalingWindowCycle{
		newAbstractCycle(name, "ScalingWindow", db, dbCollection, origin, coolDown, publishTask),
		timeWindow,
		minimumThrottle,
		maximumThrottle,
		timeWindow.String(),
		minimumThrottle.String(),
		maximumThrottle.String(),
	}
}

func (s *ScalingWindowCycle) Start() {
	log.WithField("collection", s.DBCollection).WithField("name", s.Name).WithField("coolDown", s.CoolDown).WithField("timeWindow", s.TimeWindow).Info("Starting scaling window cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.Metadata().UpdateState(startingState)
	go s.start(ctx)
}

func (s *ScalingWindowCycle) start(ctx context.Context) {
	endTime := time.Now()
	startTime := endTime.Add(-1 * s.timeWindow)

	for {
		uuidCollection, err := native.NewNativeUUIDCollectionForTimeWindow(s.db, s.DBCollection, startTime, endTime)
		if err != nil {
			log.WithError(err).WithField("start", startTime).WithField("end", endTime).Warn("Failed to query native collection for time window.")
			s.Metadata().UpdateState(coolDownState)
			time.Sleep(s.coolDown)
			endTime = time.Now()
			continue
		}

		copiedTime := startTime // Copy so that we don't change the time for the cycle
		s.CycleMetadata = &CycleMetadata{State: runningState, Iteration: s.CycleMetadata.Iteration + 1, Total: uuidCollection.Length(), Start: &copiedTime, End: &endTime, lock: &sync.RWMutex{}}
		startTime = endTime

		if uuidCollection.Length() == 0 {
			s.CycleMetadata.UpdateState(coolDownState)
			time.Sleep(s.coolDown)
			endTime = time.Now()
			continue
		}

		t, cancel := NewCappedDynamicThrottle(s.timeWindow, s.minimumThrottle, s.maximumThrottle, uuidCollection.Length()+1, 1) // add one to the length to increase the wait time
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

		endTime = time.Now()
	}

	s.Metadata().UpdateState(stoppedState)
}

func (s *ScalingWindowCycle) UpdateConfiguration() {

}

func (s *ScalingWindowCycle) TransformToConfig() *CycleConfig {
	return &CycleConfig{Name: s.Name, Type: s.Type, Collection: s.DBCollection, TimeWindow: s.TimeWindow, CoolDown: s.CoolDown, MinimumThrottle: s.MinimumThrottle, MaximumThrottle: s.MaximumThrottle}
}
