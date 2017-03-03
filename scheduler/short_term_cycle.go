package scheduler

import (
	"time"

	"github.com/Financial-Times/publish-carousel/native"
)

type ShortTermCycle struct {
	abstractCycle
	duration time.Duration
	mongo    native.DB
}

// func NewShortTermCycle() Cycle {
//
// }

// func (s *ShortTermCycle) Start() {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	s.cancel = cancel
// 	go s.start(ctx)
// }

// func (s *ShortTermCycle) start(ctx context.Context) {
// 	iterationStartTime = time.Now()
// 	for {
// 		s.collection = NewNativeUUIDCollectionForTimeWindow
// 		//update collection
// 		//s.collection
// 		//change Throttle
// 		//s.throttle =
//
// 		s.beginRun(ctx)
// 	}
// }
