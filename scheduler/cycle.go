package scheduler

import (
	"context"
	"sync"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type Cycle interface {
	Start()
	Pause()
	Resume()
	Stop()
	State() interface{}
	UpdateConfiguration()
}

var cycles = map[string]Cycle{}

type cycleState struct {
	CurrentUUID string  `json:"currentUuid"`
	Errors      int     `json:"errors"`
	Progress    float64 `json:"progress"`
	Completed   int     `json:"completed"`
	Total       int     `json:"total"`
	Iteration   int     `json:"iteration"`
	lock        *sync.RWMutex
}

type abstractCycle struct {
	Name         string      `json:"name"`
	CycleState   *cycleState `json:"state"`
	pauseLock    *sync.Mutex
	cancel       context.CancelFunc
	db           native.DB
	dbCollection string
	publishTask  tasks.Task
}

func GetCycles() map[string]Cycle {
	return cycles
}

func newAbstractCycle(name string, database native.DB, dbCollection string, task tasks.Task) *abstractCycle {
	return &abstractCycle{
		Name:         name,
		CycleState:   &cycleState{lock: &sync.RWMutex{}},
		pauseLock:    &sync.Mutex{},
		db:           database,
		dbCollection: dbCollection,
		publishTask:  task,
	}
}

func (a *abstractCycle) publishCollection(ctx context.Context, collection native.UUIDCollection, t Throttle) error {
	for {
		if err := ctx.Err(); err != nil {
			collection.Close()
			return err
		}
		a.pauseLock.Lock()

		uuid := collection.Next()
		log.WithField("uuid", uuid).Info("Running publish task.")

		t.Queue()
		err := a.publishTask.Publish(uuid)
		a.updateState(uuid, err)
		a.pauseLock.Unlock()

		if collection.Done() {
			break
		}
	}
	return nil
}

func (a *abstractCycle) updateState(uuid string, err error) {
	a.CycleState.lock.Lock()
	defer a.CycleState.lock.Unlock()

	if err != nil {
		a.CycleState.Errors++
	}

	a.CycleState.Completed++
	a.CycleState.CurrentUUID = uuid

	if a.CycleState.Total == 0 {
		a.CycleState.Progress = 0
	} else {
		a.CycleState.Progress = float64(a.CycleState.Completed) / float64(a.CycleState.Total)
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

func (a *abstractCycle) State() interface{} {
	return a.CycleState
}
