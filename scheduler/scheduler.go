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

// Scheduler is the main component of the publish carousel,
// which handles the publish cycles.
type Scheduler interface {
	Cycles() map[string]Cycle
	NewCycle(config CycleConfig) (Cycle, error)
	AddCycle(cycle Cycle) error
	DeleteCycle(cycleID string) error
	RestorePreviousState()
	Start() error
	Shutdown() error
	ManualToggleHandler(toggleValue string)
	AutomaticToggleHandler(toggleValue string)
	IsRunning() bool
	IsEnabled() bool
	IsAutomaticallyDisabled() bool
	WasAutomaticallyDisabled() bool
}

type defaultScheduler struct {
	publishTask        tasks.Task
	database           native.DB
	cycles             map[string]Cycle
	metadataReadWriter MetadataReadWriter
	cycleLock          *sync.RWMutex
	state              *schedulerState
	autoDisabled       bool
	toggleHandlerLock  *sync.Mutex
	defaultThrottle    time.Duration
}

// NewScheduler returns a new instance of the cycles scheduler
func NewScheduler(database native.DB, publishTask tasks.Task, metadataReadWriter MetadataReadWriter, defaultThrottle time.Duration) Scheduler {
	return &defaultScheduler{
		database:           database,
		publishTask:        publishTask,
		cycles:             map[string]Cycle{},
		metadataReadWriter: metadataReadWriter,
		cycleLock:          &sync.RWMutex{},
		state:              newSchedulerState(),
		toggleHandlerLock:  &sync.Mutex{},
		defaultThrottle:    defaultThrottle,
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

	if s.state.isEnabled() && s.state.isRunning() {
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

func (s *defaultScheduler) saveCycleMetadata() {
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
			cycle.SetMetadata(state)
		}
	}
}

func (s *defaultScheduler) Start() error {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()

	if !s.state.isEnabled() {
		return errors.New("Scheduler is not enabled")
	}

	if s.state.isRunning() {
		return errors.New("Scheduler is already running")
	}

	s.state.setState(running)

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

	if !s.state.isRunning() {
		return errors.New("Scheduler has already been shut down")
	}

	for id, cycle := range s.cycles {
		log.WithField("id", id).Info("Stopping cycle.")
		cycle.Stop()
	}
	s.state.setState(stopped)
	s.saveCycleMetadata()
	return nil
}

const (
	automatic = iota
	manual
)

func (s *defaultScheduler) AutomaticToggleHandler(toggleValue string) {
	s.toggleHandler(toggleValue, automatic)
}

func (s *defaultScheduler) ManualToggleHandler(toggleValue string) {
	s.toggleHandler(toggleValue, manual)
}

const (
	on  = true
	off = false
)

func (s *defaultScheduler) toggleHandler(toggleValue string, requestType int) {
	s.toggleHandlerLock.Lock()
	defer s.toggleHandlerLock.Unlock()

	toggleState, err := strconv.ParseBool(toggleValue)
	if err != nil {
		log.WithError(err).Error("Invalid toggle value for carousel scheduler")
	}

	if toggleState == off && s.state.isEnabled() {
		if s.state.isRunning() {
			log.Info("Disabling carousel scheduler...")
			err := s.Shutdown()
			if err != nil {
				log.WithError(err).Error("Error in stopping carousel scheduler")
				return
			}
		}
		if requestType == automatic {
			s.state.setState(autoDisabled)
		} else {
			s.state.setState(disabled)
		}
	} else if toggleState == on &&
		!s.state.isRunning() &&
		((s.state.isAutomaticallyDisabled() && requestType == automatic) ||
			(!s.state.isAutomaticallyDisabled() && requestType == manual)) {
		s.state.setState(stopped)
	}

}

func (s *defaultScheduler) IsEnabled() bool {
	return s.state.isEnabled()
}

func (s *defaultScheduler) IsRunning() bool {
	return s.state.isRunning()
}

func (s *defaultScheduler) IsAutomaticallyDisabled() bool {
	return s.state.isAutomaticallyDisabled()
}

func (s *defaultScheduler) WasAutomaticallyDisabled() bool {
	return s.state.wasAutomaticallyDisabled()
}

const (
	unknown = iota
	running
	stopped
	disabled
	autoDisabled
)

type schedulerState struct {
	sync.RWMutex
	currentState  int
	previousState int
}

func newSchedulerState() *schedulerState {
	return &schedulerState{currentState: disabled, previousState: unknown}
}

func (s *schedulerState) setState(stateValue int) {
	s.Lock()
	defer s.Unlock()
	if s.currentState != stateValue {
		s.previousState = s.currentState
		s.currentState = stateValue
	}
}

func (s *schedulerState) isEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.currentState != disabled && s.currentState != autoDisabled
}

func (s *schedulerState) isRunning() bool {
	s.RLock()
	defer s.RUnlock()
	return s.currentState == running
}

func (s *schedulerState) isAutomaticallyDisabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.currentState == autoDisabled
}

func (s *schedulerState) wasAutomaticallyDisabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.previousState == autoDisabled
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
		var throttleInterval time.Duration
		if config.Throttle == "" {
			log.WithField("cycleName", config.Name).Info("Throttle configuration not found. Setting default throttle value")
			throttleInterval = s.defaultThrottle
		} else {
			throttleInterval, _ = time.ParseDuration(config.Throttle)
		}
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
