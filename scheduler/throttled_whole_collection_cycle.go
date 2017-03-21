package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type ThrottledWholeCollectionCycle struct {
	*abstractCycle
	throttle Throttle
}

func NewThrottledWholeCollectionCycle(name string, db native.DB, dbCollection string, origin string, coolDown time.Duration, throttle Throttle, publishTask tasks.Task) Cycle {
	return &ThrottledWholeCollectionCycle{newAbstractCycle(name, "ThrottledWholeCollection", db, dbCollection, origin, coolDown, publishTask), throttle}
}

func (l *ThrottledWholeCollectionCycle) Start() {
	log.WithField("id", l.ID).WithField("name", l.Name).WithField("collection", l.DBCollection).Info("Starting throttled whole collection cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.Metadata().UpdateState(startingState)
	go l.start(ctx)
}

func (l *ThrottledWholeCollectionCycle) start(ctx context.Context) {
	skip := 0
	if l.CycleMetadata != nil {
		skip = l.CycleMetadata.Completed
	}

	for {
		uuidCollection, err := native.NewNativeUUIDCollection(l.db, l.DBCollection, skip)
		if err != nil {
			log.WithField("id", l.ID).WithField("name", l.Name).WithField("collection", l.DBCollection).WithError(err).Warn("Failed to consume UUIDs from the Native UUID Collection.")
			l.Metadata().UpdateState(unhealthyState, coolDownState)
			time.Sleep(l.coolDown)
			skip = l.CycleMetadata.Completed
			continue
		}

		l.CycleMetadata = &CycleMetadata{Completed: skip, State: []string{runningState}, Iteration: l.CycleMetadata.Iteration + 1, Total: uuidCollection.Length(), lock: &sync.RWMutex{}, state: make(map[string]struct{})}
		if uuidCollection.Length() == 0 {
			l.Metadata().UpdateState(unhealthyState) // assume unhealthy, as the whole archive should *always* have content
			break
		}

		stopped, err := l.publishCollection(ctx, uuidCollection, l.throttle)
		if stopped {
			break
		}

		if err != nil {
			log.WithField("id", l.ID).WithField("name", l.Name).WithField("collection", l.DBCollection).WithError(err).Error("Unexpected error occurred while publishing collection.")
			l.Metadata().UpdateState(unhealthyState)
			break
		}

		skip = 0
	}

	l.Metadata().UpdateState(stoppedState)
}

func (s *ThrottledWholeCollectionCycle) TransformToConfig() *CycleConfig {
	return &CycleConfig{Name: s.Name, Type: s.Type, Collection: s.DBCollection}
}
