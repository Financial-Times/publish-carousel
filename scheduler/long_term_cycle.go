package scheduler

import (
	"context"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
)

type LongTermCycle struct {
	*abstractCycle
}

func NewLongTermCycle(db native.DB, dbCollection string, throttle Throttle, publishTask tasks.Task) Cycle {
	return &LongTermCycle{newAbstractCycle(db, dbCollection, throttle, publishTask)}
}

func (l *LongTermCycle) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	go l.start(ctx)
}

func (l *LongTermCycle) start(ctx context.Context) {
	for {
		uuidCollection, err := native.NewNativeUUIDCollection(l.db, l.dbCollection)
		if err != nil {
			break
		}
		l.publishCollection(ctx, uuidCollection, l.throttle)
	}
}

func (l *LongTermCycle) State() interface{} {
	//TODO to implement
	return struct{}{}
}

func (l *LongTermCycle) UpdateConfiguration() {

}
