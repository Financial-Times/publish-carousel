package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"
)

const running = true
const stopped = false
const disabled = false

// Scheduler is the main component of the publish carousel,
// which handles the publish cycles.
type Scheduler interface {
	Cycles() map[string]Cycle
	NewCycle(config CycleConfig) (Cycle, error)
	AddCycle(cycle Cycle) error
	DeleteCycle(cycleID string) error
	RestorePreviousState()
	SaveCycleMetadata()
	Start() error
	Shutdown() error
	ToggleHandler(toggleValue string)
	IsRunning() bool
	IsEnabled() bool
}

type defaultScheduler struct {
	publishTask        tasks.Task
	database           native.DB
	cycles             map[string]Cycle
	metadataReadWriter MetadataReadWriter
	cycleLock          *sync.RWMutex
	executionStateLock *sync.RWMutex
	toggleLock         *sync.RWMutex

	executionState bool
	toggle         bool
}

// NewScheduler returns a new instance of the cycles scheduler
func NewScheduler(database native.DB, publishTask tasks.Task, metadataReadWriter MetadataReadWriter) Scheduler {
	return &defaultScheduler{
		database:           database,
		publishTask:        publishTask,
		cycles:             map[string]Cycle{},
		metadataReadWriter: metadataReadWriter,
		cycleLock:          &sync.RWMutex{},
		executionStateLock: &sync.RWMutex{},
		toggleLock:         &sync.RWMutex{},
		executionState:     stopped,
		toggle:             disabled,
	}
}

func (s *defaultScheduler) Cycles() map[string]Cycle {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()
	return s.cycles
}

func (s *defaultScheduler) AddCycle(c Cycle) error {
	if _, ok := s.cycles[c.ID()]; ok {
		return fmt.Errorf("Conflicting ID found for cycle %v", c.ID())
	}

	s.cycleLock.Lock()
	defer s.cycleLock.Unlock()
	s.cycles[c.ID()] = c

	if s.IsEnabled() && s.IsRunning() {
		c.Start()
	}
	return nil
}

func (s *defaultScheduler) DeleteCycle(cycleID string) error {
	s.cycleLock.Lock()
	defer s.cycleLock.Unlock()

	c, ok := s.cycles[cycleID]
	if !ok {
		return fmt.Errorf("Cannot stop cycle: cycle with id %v not found", cycleID)
	}

	c.Stop()
	delete(s.cycles, cycleID)
	return nil
}

func (s *defaultScheduler) SaveCycleMetadata() {
	for _, cycle := range s.cycles {
		switch cycle.(type) {
		case *ThrottledWholeCollectionCycle:
			s.metadataReadWriter.WriteMetadata(cycle.ID(), cycle)
		}
	}
}

func (s *defaultScheduler) RestorePreviousState() {
	s.cycleLock.Lock()
	defer s.cycleLock.Unlock()

	for id, cycle := range s.cycles {
		switch cycle.(type) {
		case *ThrottledWholeCollectionCycle:
			state, err := s.metadataReadWriter.LoadMetadata(id)
			if err != nil {
				log.WithError(err).Warn("Failed to retrieve carousel state from S3 - starting from initial state.")
				continue
			}

			log.WithField("id", cycle.ID()).WithField("iteration", state.Iteration).WithField("completed", state.Completed).Info("Restoring state for cycle.")
			cycle.RestoreMetadata(state)
		}
	}
}

func (s *defaultScheduler) Start() error {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()

	if !s.IsEnabled() {
		return errors.New("Scheduler is not enabled")
	}

	if s.IsRunning() {
		return errors.New("Scheduler is already running")
	}

	s.setCurrentExecutionState(running)

	for id, cycle := range s.cycles {
		log.WithField("id", id).Info("Starting cycle.")
		cycle.Start()
	}
	return nil
}

func (s *defaultScheduler) Shutdown() error {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()
	log.Info("Scheduler shutdown initiated.")

	if !s.IsRunning() {
		return errors.New("Scheduler has already been shut down")
	}

	for id, cycle := range s.cycles {
		log.WithField("id", id).Info("Stopping cycle.")
		cycle.Stop()
	}
	s.setCurrentExecutionState(stopped)
	return nil
}

func (s *defaultScheduler) ToggleHandler(toggleValue string) {
	toggleState, err := strconv.ParseBool(toggleValue)
	if err != nil {
		log.WithError(err).Error("Invalid toggle value for carousel scheduler")
	}

	if toggleState == disabled && s.IsEnabled() && s.IsRunning() {
		log.Info("Disabling carousel scheduler...")
		err := s.Shutdown()
		if err != nil {
			log.WithError(err).Error("Error in stopping carousel scheduler")
			return
		}
		s.SaveCycleMetadata()
	}

	s.setToggleState(toggleState)
}

func (s *defaultScheduler) IsEnabled() bool {
	s.toggleLock.RLock()
	defer s.toggleLock.RUnlock()
	return s.toggle
}

func (s *defaultScheduler) setToggleState(state bool) {
	s.toggleLock.Lock()
	defer s.toggleLock.Unlock()
	s.toggle = state
}

func (s *defaultScheduler) IsRunning() bool {
	s.executionStateLock.RLock()
	defer s.executionStateLock.RUnlock()
	return s.executionState
}

func (s *defaultScheduler) setCurrentExecutionState(state bool) {
	s.executionStateLock.Lock()
	defer s.executionStateLock.Unlock()
	s.executionState = state
}

func (s *defaultScheduler) NewCycle(config CycleConfig) (Cycle, error) {
	err := config.Validate()
	if err != nil {
		return nil, err
	}

	var c Cycle
	coolDown, _ := time.ParseDuration(config.CoolDown)

	switch strings.ToLower(config.Type) {
	case "throttledwholecollection":
		throttleInterval, _ := time.ParseDuration(config.Throttle)
		t, _ := NewThrottle(throttleInterval, 1)
		c = NewThrottledWholeCollectionCycle(config.Name, s.database, config.Collection, config.Origin, coolDown, t, s.publishTask)

	case "fixedwindow":
		interval, _ := time.ParseDuration(config.TimeWindow)
		minimumThrottle, _ := time.ParseDuration(config.MinimumThrottle)
		c = NewFixedWindowCycle(config.Name, s.database, config.Collection, config.Origin, coolDown, interval, minimumThrottle, s.publishTask)

	case "scalingwindow":
		timeWindow, _ := time.ParseDuration(config.TimeWindow)
		minimumThrottle, _ := time.ParseDuration(config.MinimumThrottle)
		maximumThrottle, _ := time.ParseDuration(config.MaximumThrottle)
		c = NewScalingWindowCycle(config.Name, s.database, config.Collection, config.Origin, timeWindow, coolDown, minimumThrottle, maximumThrottle, s.publishTask)
	}

	return c, nil
}
