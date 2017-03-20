package scheduler

import (
	"context"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type ScalingWindowCycle struct {
	*abstractTimeWindowedCycle
	maximumThrottle time.Duration
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

	base := newAbstractCycle(name, "ScalingWindow", db, dbCollection, origin, coolDown, publishTask)
	return &ScalingWindowCycle{
		newAbstractTimeWindowedCycle(base, timeWindow, minimumThrottle),
		maximumThrottle,
		maximumThrottle.String(),
	}
}

func (s *ScalingWindowCycle) Start() {
	log.WithField("collection", s.DBCollection).WithField("name", s.Name).WithField("coolDown", s.CoolDown).WithField("timeWindow", s.TimeWindow).Info("Starting scaling window cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.Metadata().UpdateState(startingState)

	throttle := func(publishes int) (Throttle, context.CancelFunc) {
		return NewCappedDynamicThrottle(s.timeWindow, s.minimumThrottle, s.maximumThrottle, publishes, 1)
	}
	go s.start(ctx, throttle)
}

func (s *ScalingWindowCycle) TransformToConfig() *CycleConfig {
	return &CycleConfig{Name: s.Name, Type: s.Type, Collection: s.DBCollection, TimeWindow: s.TimeWindow, CoolDown: s.CoolDown, MinimumThrottle: s.MinimumThrottle, MaximumThrottle: s.MaximumThrottle}
}
