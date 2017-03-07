package scheduler

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
)

type Scheduler interface {
	Cycles() map[string]Cycle
	Throttles() map[string]Throttle
	AddThrottle(name string, throttleInterval string) error
	DeleteThrottle(name string) error
	AddCycle(config CycleConfig) error
	DeleteCycle(cycleID string) error
}

type defaultScheduler struct {
	publishTask tasks.Task
	database    native.DB
	cycles      map[string]Cycle
	throttles   map[string]Throttle
	cycleLock   *sync.RWMutex
}

func NewScheduler(database native.DB, publishTask tasks.Task) Scheduler {
	return &defaultScheduler{
		database:    database,
		publishTask: publishTask,
		cycles:      map[string]Cycle{},
		throttles:   map[string]Throttle{},
		cycleLock:   &sync.RWMutex{},
	}
}

func (s *defaultScheduler) Cycles() map[string]Cycle {
	s.cycleLock.RLock()
	defer s.cycleLock.RUnlock()
	return s.cycles
}

func (s *defaultScheduler) Throttles() map[string]Throttle {
	return s.throttles
}

func (s *defaultScheduler) AddCycle(config CycleConfig) error {
	err := config.validate()
	if err != nil {
		return err
	}

	var c Cycle
	switch strings.ToLower(config.Type) {
	case "longterm":
		t, ok := s.Throttles()[config.Throttle]
		if !ok {
			return fmt.Errorf("Throttle not found for cycle %v", config.Name)
		}
		c = NewLongTermCycle(config.Name, s.database, config.Collection, t, s.publishTask)
	case "shortterm":
		interval, _ := time.ParseDuration(config.TimeWindow)
		c = NewShortTermCycle(config.Name, s.database, config.Collection, interval, s.publishTask)
	}
	if _, ok := s.cycles[c.ID()]; ok {
		return fmt.Errorf("Conflicting ID found for cycle %v", config.Name)
	}

	s.cycleLock.Lock()
	defer s.cycleLock.Unlock()

	s.cycles[c.ID()] = c
	c.Start()
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
	s.throttles[name] = t

	return nil
}

func (s *defaultScheduler) DeleteThrottle(name string) error {
	t, ok := s.throttles[name]
	if !ok {
		return fmt.Errorf("Cannot delete throttle: throttle with name %v not found", name)
	}
	t.Stop()
	delete(s.throttles, name)
	return nil
}
