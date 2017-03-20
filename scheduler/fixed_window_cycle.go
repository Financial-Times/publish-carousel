package scheduler

import (
	"context"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type FixedWindowCycle struct {
	*abstractTimeWindowedCycle
}

func NewFixedWindowCycle(name string, db native.DB, dbCollection string, origin string, coolDown time.Duration, timeWindow time.Duration, minimumThrottle time.Duration, publishTask tasks.Task) Cycle {
	basis := newAbstractCycle(name, "FixedWindow", db, dbCollection, origin, coolDown, publishTask)

	return &FixedWindowCycle{
		newAbstractTimeWindowedCycle(basis, timeWindow, minimumThrottle),
	}
}

func (s *FixedWindowCycle) Start() {
	log.WithField("collection", s.DBCollection).WithField("name", s.Name).WithField("timeWindow", s.timeWindow).Info("Starting fixed window cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.Metadata().UpdateState(startingState)

	throttle := func(publishes int) (Throttle, context.CancelFunc) {
		return NewDynamicThrottle(s.minimumThrottle, s.timeWindow, publishes, 1)
	}
	go s.start(ctx, throttle)
}

func (s *FixedWindowCycle) TransformToConfig() *CycleConfig {
	return &CycleConfig{Name: s.Name, Type: s.Type, Collection: s.DBCollection, TimeWindow: s.TimeWindow}
}
