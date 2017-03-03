package scheduler

import (
	"context"
	"sync"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
)

type LongTermCycle struct {
	collection native.UUIDCollection
	task       tasks.Task
	throttle   Throttle
	cancel     context.CancelFunc
	lock       sync.Mutex
}

func (l *LongTermCycle) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	go l.start(ctx)
}

func (l *LongTermCycle) Pause() {
	l.lock.Lock()
}

func (l *LongTermCycle) Resume() {
	l.lock.Unlock()
}

func (l *LongTermCycle) Stop() {
	l.cancel()
}

func (l *LongTermCycle) beginRun(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			break
		}

		l.lock.Lock()
		uuid := l.collection.Next()
		l.throttle.Queue()
		l.task.Publish(uuid)

		l.lock.Unlock()
	}
}

func (l *LongTermCycle) start(ctx context.Context) {
	for {
		l.beginRun(ctx)
	}
}
