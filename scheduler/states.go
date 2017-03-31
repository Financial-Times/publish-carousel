package scheduler

import (
	"encoding/json"
	"sort"
	"sync"
)

const startingState = "starting"
const runningState = "running"
const stoppedState = "stopped"
const unhealthyState = "unhealthy"
const coolDownState = "cooldown"

type State struct {
	states []string

	stateSet map[string]struct{}
	lock     *sync.RWMutex
}

func NewState() *State {
	return &State{lock: &sync.RWMutex{}, stateSet: make(map[string]struct{})}
}

func (s *State) MarshalJSON() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return json.Marshal(s.states)
}

func (s *State) UnmarshalJSON(b []byte) error {
	var states []string
	err := json.Unmarshal(b, &states)
	if err != nil {
		return err
	}

	s.Update(states...)
	return nil
}

func (s *State) Update(states ...string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stateSet = make(map[string]struct{})

	for _, state := range states {
		s.stateSet[state] = struct{}{}
	}

	var arr []string
	for k := range s.stateSet {
		arr = append(arr, k)
	}

	sort.Strings(arr)
	s.states = arr
}
