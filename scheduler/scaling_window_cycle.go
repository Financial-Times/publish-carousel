package scheduler

import (
	"context"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/sirupsen/logrus"
)

type ScalingWindowCycle struct {
	*abstractTimeWindowedCycle
	maximumThrottle time.Duration
	MaximumThrottle string `json:"maximumThrottle"`
}

func NewScalingWindowCycle(
	name string,
	uuidCollectionBuilder *native.NativeUUIDCollectionBuilder,
	dbCollection string,
	origin string,
	timeWindow time.Duration,
	coolDown time.Duration,
	minimumThrottle time.Duration,
	maximumThrottle time.Duration,
	publishTask tasks.Task,
) Cycle {

	base := newAbstractCycle(name, "ScalingWindow", uuidCollectionBuilder, dbCollection, origin, coolDown, publishTask)
	return &ScalingWindowCycle{
		newAbstractTimeWindowedCycle(base, timeWindow, minimumThrottle, maximumThrottle),
		maximumThrottle,
		maximumThrottle.String(),
	}
}

func (s *ScalingWindowCycle) Start() {
	log.WithField("id", s.CycleID).WithField("name", s.CycleName).WithField("collection", s.DBCollection).WithField("coolDown", s.CoolDown).WithField("timeWindow", s.TimeWindow).Info("Starting scaling window cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.UpdateState(startingState)

	throttle := func(publishes int) (Throttle, context.CancelFunc) {
		return NewCappedDynamicThrottle(s.timeWindow, s.minimumThrottle, s.maximumThrottle, publishes, 1)
	}
	go s.start(ctx, throttle)
}

func (s *ScalingWindowCycle) TransformToConfig() CycleConfig {
	return CycleConfig{Name: s.CycleName, Type: s.CycleType, Collection: s.DBCollection, TimeWindow: s.TimeWindow, CoolDown: s.CoolDown, MinimumThrottle: s.MinimumThrottle, MaximumThrottle: s.MaximumThrottle}
}
