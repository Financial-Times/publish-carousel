package scheduler

import (
	"context"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/sirupsen/logrus"
)

const (
	ThrottledWholeCollectionType = "ThrottledWholeCollection"
)

type ThrottledWholeCollectionCycle struct {
	*abstractCycle
	Throttle Throttle `json:"throttle"`
}

func NewThrottledWholeCollectionCycle(name string, uuidCollectionBuilder *native.NativeUUIDCollectionBuilder, dbCollection string, origin string, coolDown time.Duration, throttle Throttle, publishTask tasks.Task) Cycle {
	return &ThrottledWholeCollectionCycle{newAbstractCycle(name, ThrottledWholeCollectionType, uuidCollectionBuilder, dbCollection, origin, coolDown, publishTask), throttle}
}

func (l *ThrottledWholeCollectionCycle) Start() {
	log.WithField("id", l.CycleID).WithField("name", l.CycleName).WithField("collection", l.DBCollection).Info("Starting throttled whole collection cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.UpdateState(startingState)
	go l.start(ctx)
}

func (l *ThrottledWholeCollectionCycle) start(ctx context.Context) {
	skip := l.PublishedItems()

	b := true
	for b {
		skip, b = l.publishCollectionCycle(ctx, skip)
	}
}

func (l *ThrottledWholeCollectionCycle) publishCollectionCycle(ctx context.Context, skip int) (int, bool) {
	uuidCollection, err := l.uuidCollectionBuilder.NewNativeUUIDCollection(ctx, l.DBCollection, skip)

	if err != nil {
		log.WithField("id", l.CycleID).WithField("name", l.CycleName).WithField("collection", l.DBCollection).WithError(err).Warn("Failed to consume UUIDs from the Native UUID Collection.")
		l.UpdateState(stoppedState, unhealthyState)
		return skip, false
	}

	iteration := l.CycleMetadata.Iteration
	if skip == 0 {
		iteration++
	}

	metadata := CycleMetadata{Completed: skip, State: []string{runningState}, Iteration: iteration, Attempts: l.CycleMetadata.Attempts + 1, Total: uuidCollection.Length()}
	l.SetMetadata(metadata)

	if uuidCollection.Length() == 0 {
		l.UpdateState(stoppedState, unhealthyState) // assume unhealthy, as the whole archive should *always* have content
		return skip, false
	}

	stopped, err := l.publishCollection(ctx, uuidCollection, l.Throttle)
	if stopped {
		l.UpdateState(stoppedState)
		return skip, false
	}

	if err != nil {
		log.WithField("id", l.CycleID).WithField("name", l.CycleName).WithField("collection", l.DBCollection).WithError(err).Error("Unexpected error occurred while publishing collection.")
		l.UpdateState(stoppedState, unhealthyState)
		return skip, false
	}

	return 0, true
}

func (s *ThrottledWholeCollectionCycle) TransformToConfig() CycleConfig {
	return CycleConfig{Name: s.CycleName, Type: s.CycleType, Collection: s.DBCollection, CoolDown: s.CoolDown, Origin: s.Origin, Throttle: s.Throttle.Interval().String()}
}
