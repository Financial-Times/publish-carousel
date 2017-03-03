package scheduler

import (
	"context"
	"sync"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
)

type Cycle interface {
	Start()
	Pause()
	Resume()
	Stop()
	State() interface{}
	UpdateConfiguration()
}

//func NewCycle(throttle Throttle) Cycle {

//}

type abstractCycle struct {
	pauseLock    *sync.Mutex
	cancel       context.CancelFunc
	db           native.DB
	dbCollection string
	publishTask  tasks.Task
	throttle     Throttle
}

func newAbstractCycle(database native.DB, dbCollection string, throttle Throttle, task tasks.Task) *abstractCycle {
	return &abstractCycle{
		pauseLock:    &sync.Mutex{},
		db:           database,
		dbCollection: dbCollection,
		publishTask:  task,
		throttle:     throttle,
	}
}

func (a *abstractCycle) Pause() {
	a.pauseLock.Lock()
}

func (a *abstractCycle) Resume() {
	a.pauseLock.Unlock()
}

func (a *abstractCycle) Stop() {
	a.cancel()
}
