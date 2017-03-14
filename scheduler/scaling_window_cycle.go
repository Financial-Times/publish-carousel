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
	coolDown        time.Duration
	minimumThrottle time.Duration
	maximumThrottle time.Duration
	TimeWindow      string `json:"timeWindow"`
	CoolDown        string `json:"coolDown"`
	MinimumThrottle string `json:"minimumThrottle"`
	MaximumThrottle string `json:"maximumThrottle"`
}

func NewScalingWindowCycle(name string, db native.DB, dbCollection string, timeWindow time.Duration, coolDown time.Duration, minimumThrottle time.Duration, maximumThrottle time.Duration, publishTask tasks.Task) Cycle {
	return &ScalingWindowCycle{
		newAbstractCycle(name, "ScalingWindow", db, dbCollection, publishTask),
		timeWindow,
		coolDown,
		minimumThrottle,
		maximumThrottle,
		timeWindow.String(),
		coolDown.String(),
		minimumThrottle.String(),
		maximumThrottle.String(),
	}
}

func (s *ScalingWindowCycle) Start() {
	log.WithField("collection", s.DBCollection).WithField("name", s.Name).WithField("coolDown", s.CoolDown).WithField("timeWindow", s.TimeWindow).Info("Starting scaling window cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.start(ctx)
}

func (s *ScalingWindowCycle) start(ctx context.Context) {
	startTime := time.Now().Add(-1 * s.timeWindow)
	for {
		endTime := startTime.Add(s.timeWindow)
		uuidCollection, err := native.NewNativeUUIDCollectionForTimeWindow(s.db, s.DBCollection, startTime, endTime)
		if err != nil {
			log.WithError(err).WithField("start", startTime).WithField("end", endTime).Warn("Failed to query native collection for time window.")
			break
		}

		s.CycleMetadata = &CycleMetadata{State: runningState, Iteration: s.CycleMetadata.Iteration + 1, Total: uuidCollection.Length(), Start: &startTime, End: &endTime, lock: &sync.RWMutex{}}
		startTime = endTime

		if uuidCollection.Length() == 0 {
			time.Sleep(s.coolDown)
			continue
		}

		t, cancel := NewCappedDynamicThrottle(s.minimumThrottle, s.timeWindow, s.maximumThrottle, uuidCollection.Length()+1, 1) // add one to the length to increase the wait time
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
}

func (s *ScalingWindowCycle) UpdateConfiguration() {

}

func (s *ScalingWindowCycle) TransformToConfig() *CycleConfig {
	return &CycleConfig{Name: s.Name, Type: s.Type, Collection: s.DBCollection, TimeWindow: s.TimeWindow, CoolDown: s.CoolDown, MinimumThrottle: s.MinimumThrottle, MaximumThrottle: s.MaximumThrottle}
}
