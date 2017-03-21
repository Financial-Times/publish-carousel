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
const enabled = true
const disabled = false

// Scheduler is the main component of the publish carousel,
// which handles the publish cycles.
type Scheduler interface {
	Cycles() map[string]Cycle
	Throttles() map[string]Throttle
	AddThrottle(name string, throttleInterval string) error
	DeleteThrottle(name string) error
	NewCycle(config CycleConfig) (Cycle, error)
	AddCycle(cycle Cycle) error
	DeleteCycle(cycleID string) error
	RestorePreviousState()
	SaveCycleMetadata()
	Start() error
	Shutdown() error
	ToggleHandler(toggleValue string)
}

type defaultScheduler struct {
	publishTask                tasks.Task
	database                   native.DB
	cycles                     map[string]Cycle
	throttles                  map[string]Throttle
	metadataReadWriter         MetadataReadWriter
	throttleLock               *sync.RWMutex
	cycleLock                  *sync.RWMutex
	currentExecutionStateLock  *sync.RWMutex
	previousExecutionStateLock *sync.RWMutex
	toggleLock                 *sync.RWMutex
	currentExecutionState      bool
	previousExecutionState     bool
	toggle                     bool
}

// NewScheduler returns a new instance of the cycles scheduler
func NewScheduler(database native.DB, publishTask tasks.Task, metadataReadWriter MetadataReadWriter) Scheduler {
	return &defaultScheduler{
		database:                   database,
		publishTask:                publishTask,
		cycles:                     map[string]Cycle{},
		throttles:                  map[string]Throttle{},
		metadataReadWriter:         metadataReadWriter,
		cycleLock:                  &sync.RWMutex{},
		throttleLock:               &sync.RWMutex{},
		currentExecutionStateLock:  &sync.RWMutex{},
		previousExecutionStateLock: &sync.RWMutex{},
		toggleLock:                 &sync.RWMutex{},
		currentExecutionState:      stopped,
		previousExecutionState:     stopped,
		toggle:                     disabled,
	}
}

func (s *defaultScheduler) Cycles() map[string]Cycle {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()
	return s.cycles
}

func (s *defaultScheduler) Throttles() map[string]Throttle {
	s.throttleLock.RLock()
	defer s.throttleLock.RUnlock()
	return s.throttles
}

func (s *defaultScheduler) AddCycle(c Cycle) error {

	if _, ok := s.cycles[c.ID()]; ok {
		return fmt.Errorf("Conflicting ID found for cycle %v", c.ID())
	}

	s.cycleLock.Lock()
	defer s.cycleLock.Unlock()
	s.cycles[c.ID()] = c

	if s.isEnabled() && s.isRunning() {
		err := s.Start()
		if err != nil {
			return fmt.Errorf("Error in starting cycle with ID: %v - %v", c.ID(), err)
		}
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

func (s *defaultScheduler) AddThrottle(name string, throttleInterval string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("Invalid throttle name")
	}

	interval, err := time.ParseDuration(throttleInterval)
	if err != nil {
		return fmt.Errorf("Error parsing throttle interval for %v: %v", name, err)
	}

	if _, ok := s.throttles[name]; ok {
		return fmt.Errorf("Conflicting throttle name: %v ", name)
	}

	t, _ := NewThrottle(interval, 1)
	s.throttleLock.Lock()
	defer s.throttleLock.Unlock()
	s.throttles[name] = t

	return nil
}

func (s *defaultScheduler) DeleteThrottle(name string) error {
	s.throttleLock.Lock()
	defer s.throttleLock.Unlock()

	t, ok := s.throttles[name]
	if !ok {
		return fmt.Errorf("Cannot delete throttle: throttle with name %v not found", name)
	}

	t.Stop()
	delete(s.throttles, name)
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

	if !s.isEnabled() {
		return errors.New("carousel scheduler is not enabled")
	}

	if s.isRunning() {
		return errors.New("carousel scheduler is already running")
	}

	s.setCurrentExecutionState(running)

	for id, cycle := range s.cycles {
		switch cycle.Metadata().State {
		case stoppedState:
			log.WithField("id", cycle.ID()).Info("Configured cycle has been stopped during the Carousel startup process - should this cycle be removed from the configuration file?")
			cycle.Start()
		default:
			log.WithField("id", id).Info("Starting cycle.")
			cycle.Start()
		}
	}
	return nil
}

func (s *defaultScheduler) Shutdown() error {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()
	log.Info("Scheduler shutdown initiated.")

	if !s.isRunning() {
		return errors.New("carousel scheduler have been already shutted down")
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
	if toggleState == disabled && s.isEnabled() && s.isRunning() {
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

func (s *defaultScheduler) isEnabled() bool {
	s.toggleLock.RLock()
	defer s.toggleLock.RUnlock()
	return s.toggle
}

func (s *defaultScheduler) setToggleState(state bool) {
	s.toggleLock.Lock()
	defer s.toggleLock.Unlock()
	s.toggle = state
}

func (s *defaultScheduler) isRunning() bool {
	s.currentExecutionStateLock.RLock()
	defer s.currentExecutionStateLock.RUnlock()
	return s.currentExecutionState
}

func (s *defaultScheduler) setCurrentExecutionState(state bool) {
	s.currentExecutionStateLock.Lock()
	defer s.currentExecutionStateLock.Unlock()
	s.currentExecutionState = state
}

func (s *defaultScheduler) NewCycle(config CycleConfig) (Cycle, error) {
	err := config.Validate()
	if err != nil {
		return nil, err
	}

	var c Cycle
	switch strings.ToLower(config.Type) {
	case "throttledwholecollection":
		t, ok := s.Throttles()[config.Throttle]
		if !ok {
			return nil, fmt.Errorf("Throttle not found for cycle %v", config.Name)
		}
		c = NewThrottledWholeCollectionCycle(config.Name, s.database, config.Collection, config.Origin, t, s.publishTask)

	case "fixedwindow":
		interval, _ := time.ParseDuration(config.TimeWindow)
		minimumThrottle, _ := time.ParseDuration(config.MinimumThrottle)
		c = NewFixedWindowCycle(config.Name, s.database, config.Collection, config.Origin, interval, minimumThrottle, s.publishTask)

	case "scalingwindow":
		timeWindow, _ := time.ParseDuration(config.TimeWindow)
		coolDown, _ := time.ParseDuration(config.CoolDown)
		minimumThrottle, _ := time.ParseDuration(config.MinimumThrottle)
		maximumThrottle, _ := time.ParseDuration(config.MaximumThrottle)
		c = NewScalingWindowCycle(config.Name, s.database, config.Collection, config.Origin, timeWindow, coolDown, minimumThrottle, maximumThrottle, s.publishTask)
	}

	return c, nil
}
