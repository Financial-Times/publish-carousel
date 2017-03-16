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

type Scheduler interface {
	Cycles() map[string]Cycle
	Throttles() map[string]Throttle
	AddThrottle(name string, throttleInterval string) error
	DeleteThrottle(name string) error
	AddCycle(config CycleConfig) error
	DeleteCycle(cycleID string) error
	RestorePreviousState()
	SaveCycleMetadata()
	Start() error
	Shutdown() error
	ToggleHandler(toggleValue string)
}

type defaultScheduler struct {
	publishTask        tasks.Task
	database           native.DB
	cycles             map[string]Cycle
	throttles          map[string]Throttle
	metadataReadWriter MetadataReadWriter
	throttleLock       *sync.RWMutex
	cycleLock          *sync.RWMutex
	isRunningLock      *sync.RWMutex
	isEnabledLock      *sync.RWMutex
	running            bool
	enabled            bool
}

func NewScheduler(database native.DB, publishTask tasks.Task, metadataReadWriter MetadataReadWriter) Scheduler {
	return &defaultScheduler{
		database:           database,
		publishTask:        publishTask,
		cycles:             map[string]Cycle{},
		throttles:          map[string]Throttle{},
		metadataReadWriter: metadataReadWriter,
		cycleLock:          &sync.RWMutex{},
		throttleLock:       &sync.RWMutex{},
		isRunningLock:      &sync.RWMutex{},
		isEnabledLock:      &sync.RWMutex{},
		running:            false,
		enabled:            false,
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

func (s *defaultScheduler) AddCycle(config CycleConfig) error {
	err := config.Validate()
	if err != nil {
		return err
	}

	var c Cycle
	switch strings.ToLower(config.Type) {
	case "throttledwholecollection":
		t, ok := s.Throttles()[config.Throttle]
		if !ok {
			return fmt.Errorf("Throttle not found for cycle %v", config.Name)
		}
		c = NewThrottledWholeCollectionCycle(config.Name, s.database, config.Collection, t, s.publishTask)

	case "fixedwindow":
		interval, _ := time.ParseDuration(config.TimeWindow)
		minimumThrottle, _ := time.ParseDuration(config.MinimumThrottle)
		c = NewFixedWindowCycle(config.Name, s.database, config.Collection, interval, minimumThrottle, s.publishTask)

	case "scalingwindow":
		timeWindow, _ := time.ParseDuration(config.TimeWindow)
		coolDown, _ := time.ParseDuration(config.CoolDown)
		minimumThrottle, _ := time.ParseDuration(config.MinimumThrottle)
		maximumThrottle, _ := time.ParseDuration(config.MaximumThrottle)
		c = NewScalingWindowCycle(config.Name, s.database, config.Collection, timeWindow, coolDown, minimumThrottle, maximumThrottle, s.publishTask)
	}

	if _, ok := s.cycles[c.ID()]; ok {
		return fmt.Errorf("Conflicting ID found for cycle %v", config.Name)
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

	s.setRunningState(true)
	return nil
}

func (s *defaultScheduler) Shutdown() error {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()
	log.Info("Scheduler shutdown initiated.")

	if !s.isRunning() {
		return errors.New("carousel scheduler have been already shutted down")
	}

	for _, cycle := range s.cycles {
		cycle.Stop()
	}

	s.setRunningState(false)
	return nil
}

func (s *defaultScheduler) ToggleHandler(toggleValue string) {
	shouldBeEnabled, err := strconv.ParseBool(toggleValue)
	if err != nil {
		log.WithError(err).Error("Invalid toggle value for carousel scheduler")
	}
	if shouldBeEnabled && !s.isEnabled() && s.isRunning() {
		s.RestorePreviousState()
		s.Start()
	}
	if !shouldBeEnabled && s.isEnabled() && s.isRunning() {
		s.Shutdown()
		s.SaveCycleMetadata()
	}

	s.setEnableState(shouldBeEnabled)
}

func (s *defaultScheduler) isEnabled() bool {
	s.isEnabledLock.RLock()
	defer s.isEnabledLock.RUnlock()
	return s.enabled
}

func (s *defaultScheduler) setEnableState(state bool) {
	s.isEnabledLock.Lock()
	defer s.isEnabledLock.Unlock()
	s.enabled = state
}

func (s *defaultScheduler) isRunning() bool {
	s.isRunningLock.RLock()
	defer s.isRunningLock.RUnlock()
	return s.running
}

func (s *defaultScheduler) setRunningState(state bool) {
	s.isRunningLock.Lock()
	defer s.isRunningLock.Unlock()
	s.running = state
}
