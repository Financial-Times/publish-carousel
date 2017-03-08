package scheduler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

type Cycle interface {
	ID() string
	Start()
	Pause()
	Resume()
	Stop()
	State() interface{}
	UpdateConfiguration()
}

type CycleState struct {
	CurrentUUID string     `json:"currentUuid"`
	Errors      int        `json:"errors"`
	Progress    float64    `json:"progress"`
	Completed   int        `json:"completed"`
	Total       int        `json:"total"`
	Iteration   int        `json:"iteration"`
	Start       *time.Time `json:"windowStart,omitempty"`
	End         *time.Time `json:"windowEnd,omitempty"`
	lock        *sync.RWMutex
}

type abstractCycle struct {
	CycleID      string      `json:"id"`
	Name         string      `json:"name"`
	CycleState   *CycleState `json:"state"`
	pauseLock    *sync.Mutex
	cancel       context.CancelFunc
	db           native.DB
	dbCollection string
	publishTask  tasks.Task
}

func newAbstractCycle(name string, database native.DB, dbCollection string, task tasks.Task) *abstractCycle {
	return &abstractCycle{
		CycleID:      newCycleID(name),
		Name:         name,
		CycleState:   &CycleState{lock: &sync.RWMutex{}},
		pauseLock:    &sync.Mutex{},
		db:           database,
		dbCollection: dbCollection,
		publishTask:  task,
	}
}

func newCycleID(name string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte(time.Now().String()))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (a *abstractCycle) publishCollection(ctx context.Context, collection native.UUIDCollection, t Throttle) error {
	for !collection.Done() {
		if err := ctx.Err(); err != nil {
			collection.Close()
			return err
		}

		a.pauseLock.Lock()

		uuid := collection.Next()
		log.WithField("uuid", uuid).Info("Running publish task.")

		t.Queue()
		err := a.publishTask.Publish(a.dbCollection, uuid)
		a.updateState(uuid, err)
		a.pauseLock.Unlock()
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

func (a *abstractCycle) ID() string {
	return a.CycleID
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
	return *a.CycleState
}
