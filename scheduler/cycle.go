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
const stoppedState = "stopped"

type Cycle interface {
	ID() string
	Start()
	Stop()
	Reset()
	Metadata() *CycleMetadata
	RestoreMetadata(state *CycleMetadata)
	TransformToConfig() *CycleConfig
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
		DBCollection:  dbCollection,
		publishTask:   task,
	}
}

type abstractCycle struct {
	CycleID       string         `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	CycleMetadata *CycleMetadata `json:"metadata"`
	DBCollection  string         `json:"collection"`
	pauseLock     *sync.Mutex
	cancel        context.CancelFunc
	db            native.DB
	publishTask   tasks.Task
}

func (a *abstractCycle) publishCollection(ctx context.Context, collection native.UUIDCollection, t Throttle) (bool, error) {
	for !collection.Done() {
		t.Queue()

		if err := ctx.Err(); err != nil {
			collection.Close()
			return true, err
		}

		uuid := collection.Next()
		log.WithField("uuid", uuid).Info("Running publish task.")

		err := a.publishTask.Publish(a.DBCollection, uuid)
		if err != nil {
			log.WithError(err).WithField("uuid", uuid).WithField("collection", a.DBCollection).Warn("Failed to publish!")
		}

		a.updateState(uuid, err)
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

func (a *abstractCycle) Stop() {
	a.cancel()
	log.WithField("id", a.ID()).Info("Cycle stopped.")
	a.CycleMetadata.State = stoppedState
}

func (a *abstractCycle) Reset() {
	a.Stop()
	a.CycleMetadata = nil
}

func (a *abstractCycle) Metadata() *CycleMetadata {
	return a.CycleMetadata
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
