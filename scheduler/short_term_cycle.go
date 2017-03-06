package scheduler

import (
	"context"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
)

type ShortTermCycle struct {
	*abstractCycle
	duration time.Duration
	mongo    native.DB
}

// func NewShortTermCycle() Cycle {
//
// }

func (s *ShortTermCycle) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.start(ctx)
}

func (s *ShortTermCycle) start(ctx context.Context) {
	startTime := time.Now().Add(-1 * s.duration)
	for {
		endTime := startTime.Add(s.duration)
		uuidCollection, err := native.NewNativeUUIDCollectionForTimeWindow(s.db, s.dbCollection, startTime, endTime)
		if err != nil {
			break
		}

		t, cancel := NewDynamicThrottle(s.duration, uuidCollection.Length(), 1)
		s.publishCollection(ctx, uuidCollection, t)
		cancel()
		startTime = endTime
	}
}
