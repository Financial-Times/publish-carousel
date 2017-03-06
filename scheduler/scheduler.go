package scheduler

type Scheduler interface {
	Cycles() map[string]Cycle
	Throttles() map[string]Throttle
	StartAllCycles()
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
