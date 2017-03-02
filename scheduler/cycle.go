package scheduler

type Cycle interface {
	Start() error
	Pause() error
	Stop() error
	State() interface{}
	UpdateConfiguration(config CycleConfig)
}

type CycleConfig struct {
}

//func NewCycle(config CycleConfig, throttle Throttle) Cycle {

//}
