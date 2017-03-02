package scheduler

import (
	"context"

	"github.com/Financial-Times/publish-carousel/tasks"
)

type LongTermCycle struct {
	collector tasks.UUIDCollector
	task      tasks.Task
	throttle  Throttle
	cancel    context.CancelFunc
}

func (l *LongTermCycle) Start() error {
	go l.start()

	return nil
}

func (l *LongTermCycle) Stop() error {
	l.cancel()

}

func (l *LongTermCycle) start(ctx context.Context) error {
	uuids := l.collector.Collect()
	for {
		uuid := <-uuids
		l.throttle.Queue()
		l.task.Do(uuid)
	}
	return nil
}
