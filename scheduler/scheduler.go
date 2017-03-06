package scheduler

type Scheduler interface {
	Cycles() map[string]Cycle
	Throttles() map[string]Throttle
	StartAllCycles()
	Add(cycle Cycle)
	Delete(cycleId string)
}

type defaultScheduler struct {
	cycles    map[string]Cycle
	throttles map[string]Throttle
}

func (s *defaultScheduler) Cycles() map[string]Cycle {
	return s.cycles
}

func (s *defaultScheduler) Throttles() map[string]Throttle {
	return s.throttles
}

func (s *defaultScheduler) StartAllCycles() {
	for _, c := range s.cycles {
		c.Start()
	}
}

func (s *defaultScheduler) Add(cycle Cycle) {
	//TODO check if it si there
	s.cycles[cycle.ID()] = cycle
}

func (s *defaultScheduler) Delete(cycleID string) {
	s.cycles[cycleID].Stop()
	delete(s.cycles, cycleID)
}
