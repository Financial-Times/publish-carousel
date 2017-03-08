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
	return &LongTermCycle{newAbstractCycle(name, db, dbCollection, publishTask), throttle}
}

func (l *LongTermCycle) Start() {
	log.WithField("name", l.Name).Info("Starting long term cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	go l.start(ctx)
}

func (l *LongTermCycle) Restore(state *CycleState) {
	log.WithField("name", l.Name).Info("Starting long term cycle.")
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.CycleState = state

	go l.start(ctx)
}

func (l *LongTermCycle) start(ctx context.Context) {
	skip := 0
	if l.CycleState != nil {
		skip = l.CycleState.Completed
	}

	for {
		uuidCollection, err := native.NewNativeUUIDCollection(l.db, l.dbCollection, skip)
		if err != nil {
			log.WithError(err).Warn("Failed to consume UUIDs from the Native UUID Collection.")
			break
		}

		l.CycleState = &CycleState{Iteration: l.CycleState.Iteration + 1, Total: uuidCollection.Length(), lock: &sync.RWMutex{}}
		l.publishCollection(ctx, uuidCollection, l.throttle)

		skip = 0
	}
}

func (l *LongTermCycle) UpdateConfiguration() {

}
