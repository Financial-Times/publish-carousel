package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type ShortTermCycle struct {
	*abstractCycle
	duration   time.Duration
	TimeWindow string `json:"timeWindow"`
}

func NewShortTermCycle(name string, db native.DB, dbCollection string, duration time.Duration, publishTask tasks.Task) Cycle {
	return &ShortTermCycle{newAbstractCycle(name, db, dbCollection, publishTask), duration, duration.String()}
}

func (s *ShortTermCycle) Start() {
	log.WithField("name", s.Name).WithField("timeWindow", s.duration).Info("Starting short term cycle.")
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
			log.WithError(err).WithField("start", startTime).WithField("end", endTime).Warn("Failed to query native collection for time window.")
			break
		}

		s.CycleState = &CycleState{Iteration: s.CycleState.Iteration + 1, Total: uuidCollection.Length(), Start: &startTime, End: &endTime, lock: &sync.RWMutex{}}
		startTime = endTime

		if uuidCollection.Length() == 0 {
			time.Sleep(s.duration)
			continue
		}

		t, cancel := NewDynamicThrottle(s.duration, uuidCollection.Length()+1, 1) // add one to the length to increase the wait time
		s.publishCollection(ctx, uuidCollection, t)
		t.Queue() // ensure we wait a reasonable amount of time before the next iteration
		cancel()
	}
}

func (s *ShortTermCycle) UpdateConfiguration() {

}
