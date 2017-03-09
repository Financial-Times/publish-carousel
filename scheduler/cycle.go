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

const runningState = "running"
const pausedState = "paused"
const stoppedState = "stopped"

type Cycle interface {
	ID() string
	Start()
	Pause()
	Resume()
	Stop()
	Metadata() CycleMetadata
	RestoreMetadata(state *CycleMetadata)
}

type CycleMetadata struct {
	CurrentUUID string     `json:"currentUuid"`
	Errors      int        `json:"errors"`
	Progress    float64    `json:"progress"`
	State       string     `json:"state"`
	Completed   int        `json:"completed"`
	Total       int        `json:"total"`
	Iteration   int        `json:"iteration"`
	Start       *time.Time `json:"windowStart,omitempty"`
	End         *time.Time `json:"windowEnd,omitempty"`
	lock        *sync.RWMutex
}

func newCycleID(name string, dbcollection string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte(dbcollection))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func newAbstractCycle(name string, cycleType string, database native.DB, dbCollection string, task tasks.Task) *abstractCycle {
	return &abstractCycle{
		CycleID:       newCycleID(name, dbCollection),
		Name:          name,
		Type:          cycleType,
		CycleMetadata: &CycleMetadata{lock: &sync.RWMutex{}},
		pauseLock:     &sync.Mutex{},
		db:            database,
		dbCollection:  dbCollection,
		publishTask:   task,
	}
}

type abstractCycle struct {
	CycleID       string         `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	CycleMetadata *CycleMetadata `json:"metadata"`
	pauseLock     *sync.Mutex
	cancel        context.CancelFunc
	db            native.DB
	dbCollection  string
	publishTask   tasks.Task
}

func (a *abstractCycle) publishCollection(ctx context.Context, collection native.UUIDCollection, t Throttle) (bool, error) {
	for !collection.Done() {
		t.Queue()

		if err := ctx.Err(); err != nil {
			collection.Close()
			log.WithField("state", a.CycleMetadata.State).Info("hi i have stopped")
			return true, err
		}

		a.pauseLock.Lock()
		uuid := collection.Next()
		log.WithField("uuid", uuid).Info("Running publish task.")

		err := a.publishTask.Publish(a.dbCollection, uuid)
		if err != nil {
			log.WithError(err).WithField("uuid", uuid).WithField("collection", a.dbCollection).Warn("Failed to publish!")
		}

		a.updateState(uuid, err)
		a.pauseLock.Unlock()
	}
	return false, nil
}

func (a *abstractCycle) updateState(uuid string, err error) {
	a.CycleMetadata.lock.Lock()
	defer a.CycleMetadata.lock.Unlock()

	if err != nil {
		a.CycleMetadata.Errors++
	}

	a.CycleMetadata.Completed++
	a.CycleMetadata.CurrentUUID = uuid

	if a.CycleMetadata.Total == 0 {
		a.CycleMetadata.Progress = 0
	} else {
		a.CycleMetadata.Progress = float64(a.CycleMetadata.Completed) / float64(a.CycleMetadata.Total)
	}
}

func (a *abstractCycle) ID() string {
	return a.CycleID
}

func (a *abstractCycle) Pause() {
	a.pauseLock.Lock()
	log.WithField("id", a.ID()).Info("Cycle paused.")

	a.CycleMetadata.State = pausedState
}

func (a *abstractCycle) Resume() {
	a.pauseLock.Unlock()
	log.WithField("id", a.ID()).Info("Cycle resumed.")
	a.CycleMetadata.State = runningState
}

func (a *abstractCycle) Stop() {
	a.cancel()
	log.WithField("id", a.ID()).Info("Cycle stopped.")
	a.CycleMetadata.State = stoppedState
}

func (a *abstractCycle) Metadata() CycleMetadata {
	return *a.CycleMetadata
}

func (a *abstractCycle) RestoreMetadata(metadata *CycleMetadata) {
	metadata.lock = &sync.RWMutex{}
	a.CycleMetadata = metadata
}

func (c *CycleMetadata) UpdateState(state string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.State = state
}
