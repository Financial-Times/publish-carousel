package scheduler

type Cycle interface {
	Start()
	Pause()
	Resume()
	Stop()
	State() interface{}
	UpdateConfiguration()
}

//func NewCycle(throttle Throttle) Cycle {

//}
