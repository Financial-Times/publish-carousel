package scheduler

import (
	"sync"
	"time"
)

type checkpointHandler struct {
	sync.Mutex
	ticker   *time.Ticker
	interval time.Duration
	stopChan chan bool
}

func newCheckpointHandler(checkpointInterval time.Duration) *checkpointHandler {
	return &checkpointHandler{
		interval: checkpointInterval,
		stopChan: make(chan bool),
	}
}

func (ch *checkpointHandler) start(checkpointFunc func()) {
	ch.Lock()
	defer ch.Unlock()

	ch.ticker = time.NewTicker(ch.interval)

	go func() {
		for {
			select {
			case <-ch.ticker.C:
				checkpointFunc()
			case <-ch.stopChan:
				return
			}
		}
	}()
}

func (ch *checkpointHandler) stop() {
	ch.Lock()
	defer ch.Unlock()
	ch.stopChan <- true
	ch.ticker.Stop()
}
