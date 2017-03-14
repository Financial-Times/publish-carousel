package scheduler

import (
	"context"
	"sync"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type ThrottledWholeCollectionCycle struct {
	*abstractCycle
	throttle Throttle
}

func NewThrottledWholeCollectionCycle(name string, db native.DB, dbCollection string, throttle Throttle, publishTask tasks.Task) Cycle {
	return &ThrottledWholeCollectionCycle{newAbstractCycle(name, "ThrottledWholeCollection", db, dbCollection, publishTask), throttle}
}

func (l *ThrottledWholeCollectionCycle) Start() {
	log.WithField("collection", l.DBCollection).WithField("name", l.Name).Info("Starting throttled whole collection cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
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
			log.WithError(err).Warn("Failed to consume UUIDs from the Native UUID Collection.")
			break
		}

		l.CycleMetadata = &CycleMetadata{State: runningState, Iteration: l.CycleMetadata.Iteration + 1, Total: uuidCollection.Length(), lock: &sync.RWMutex{}}

		stopped, err := l.publishCollection(ctx, uuidCollection, l.throttle)
		if stopped {
			break
		}

		if err != nil {
			log.WithError(err).WithField("collection", l.DBCollection).WithField("id", l.ID).Error("Unexpected error occurred while publishing collection.")
			break
		}

		skip = 0
	}
}

func (l *ThrottledWholeCollectionCycle) UpdateConfiguration() {

}

func (s *ThrottledWholeCollectionCycle) TransformToConfig() *CycleConfig {
	return &CycleConfig{Name: s.Name, Type: s.Type, Collection: s.DBCollection}
}
