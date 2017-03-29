package scheduler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

const startingState = "starting"
const runningState = "running"
const stoppedState = "stopped"
const unhealthyState = "unhealthy"
const coolDownState = "cooldown"

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
	State       []string   `json:"state"`
	Completed   int        `json:"completed"`
	Total       int        `json:"total"`
	Iteration   int        `json:"iteration"`
	Start       *time.Time `json:"windowStart,omitempty"`
	End         *time.Time `json:"windowEnd,omitempty"`

	state map[string]struct{}
	lock  *sync.RWMutex
}

func newCycleID(name string, dbcollection string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte(dbcollection))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func newAbstractCycle(name string, cycleType string, database native.DB, dbCollection string, origin string, coolDown time.Duration, task tasks.Task) *abstractCycle {
	return &abstractCycle{
		CycleID:       newCycleID(name, dbCollection),
		Name:          name,
		Type:          cycleType,
		CycleMetadata: &CycleMetadata{lock: &sync.RWMutex{}, state: make(map[string]struct{})},
		pauseLock:     &sync.Mutex{},
		db:            database,
		DBCollection:  dbCollection,
		Origin:        origin,
		CoolDown:      coolDown.String(),
		coolDown:      coolDown,
		publishTask:   task,
	}
}

type abstractCycle struct {
	CycleID       string         `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	CycleMetadata *CycleMetadata `json:"metadata"`
	DBCollection  string         `json:"collection"`
	Origin        string         `json:"origin"`
	CoolDown      string         `json:"coolDown"`

	coolDown    time.Duration
	pauseLock   *sync.Mutex
	cancel      context.CancelFunc
	db          native.DB
	publishTask tasks.Task
}

func (a *abstractCycle) publishCollection(ctx context.Context, collection native.UUIDCollection, t Throttle) (bool, error) {
	for {
		t.Queue()

		if err := ctx.Err(); err != nil {
			collection.Close()
			return true, err
		}

		finished, uuid, err := collection.Next()
		if finished {
			log.WithField("id", a.CycleID).WithField("name", a.Name).WithField("collection", a.DBCollection).Info("Finished publishing collection.")
			a.updateState("", err)
			return false, err
		}

		if strings.TrimSpace(uuid) == "" {
			log.WithField("id", a.CycleID).WithField("name", a.Name).WithField("collection", a.DBCollection).Warn("Next UUID is empty! Skipping.")
			a.updateState(uuid, errors.New("Empty uuid"))
			continue
		}

		log.WithField("id", a.CycleID).WithField("name", a.Name).WithField("collection", a.DBCollection).WithField("uuid", uuid).Info("Running publish task.")
		err = a.publishTask.Publish(a.Origin, a.DBCollection, uuid)
		if err != nil {
			log.WithField("id", a.CycleID).WithField("name", a.Name).WithField("collection", a.DBCollection).WithField("uuid", uuid).WithError(err).Warn("Failed to publish!")
		}

		a.updateState(uuid, err)
	}
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
	log.WithField("id", a.CycleID).WithField("name", a.Name).WithField("collection", a.DBCollection).Info("Cycle stopped.")
	a.Metadata().UpdateState(stoppedState)
}

func (a *abstractCycle) Reset() {
	a.Stop()
	a.CycleMetadata = &CycleMetadata{lock: &sync.RWMutex{}, state: make(map[string]struct{})}
}

func (a *abstractCycle) Metadata() *CycleMetadata {
	return a.CycleMetadata
}

func (a *abstractCycle) RestoreMetadata(metadata *CycleMetadata) {
	metadata.lock = &sync.RWMutex{}
	metadata.state = make(map[string]struct{})
	a.CycleMetadata = metadata
}

func (c *CycleMetadata) UpdateState(states ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.state = make(map[string]struct{})

	for _, state := range states {
		c.state[state] = struct{}{}
	}

	var arr []string
	for k := range c.state {
		arr = append(arr, k)
	}

	sort.Strings(arr)
	c.State = arr
}
