package scheduler

import (
	"context"
	"sync"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type LongTermCycle struct {
	*abstractCycle
	throttle Throttle
}

func NewLongTermCycle(name string, db native.DB, dbCollection string, throttle Throttle, publishTask tasks.Task) Cycle {
	return &LongTermCycle{newAbstractCycle(name, "LongTerm", db, dbCollection, publishTask), throttle}
}

func (l *LongTermCycle) Start() {
	log.WithField("name", l.Name).Info("Starting long term cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	go l.start(ctx)
}

func (l *LongTermCycle) Restore(state *CycleMetadata) {
	log.WithField("name", l.Name).Info("Starting long term cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.CycleMetadata = state

	go l.start(ctx)
}

func (l *LongTermCycle) start(ctx context.Context) {
	skip := 0
	if l.CycleMetadata != nil {
		skip = l.CycleMetadata.Completed
	}

	for {
		uuidCollection, err := native.NewNativeUUIDCollection(l.db, l.dbCollection, skip)
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
			log.WithError(err).WithField("collection", l.dbCollection).WithField("id", l.ID).Error("Unexpected error occurred while publishing collection.")
			break
		}

		skip = 0
	}
}

func (l *LongTermCycle) UpdateConfiguration() {

}
