package scheduler

import (
	"io/ioutil"
	"time"

	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

type cycleSetupConfig struct {
	Throttles       map[string]string `yaml:"throttles"`
	LongTermCycles  []cycleConfig     `yaml:"longTermCycles"`
	ShortTermCycles []cycleConfig     `yaml:"shortTermCycles"`
}

type cycleConfig struct {
	Name       string `yaml:"name"`
	Collection string `yaml:"collection"`
	Throttle   string `yaml:"throttle"`
	TimeWindow string `yaml:"timeWindow"`
}

func LoadSchedulerFromFile(mongo native.DB, configFile string) (Scheduler, error) {
	scheduler := &defaultScheduler{throttles: map[string]Throttle{}, cycles: map[string]Cycle{}}

	fileData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return scheduler, err
	}

	setup := cycleSetupConfig{}
	err = yaml.Unmarshal(fileData, &setup)
	if err != nil {
		return scheduler, err
	}

	for name, duration := range setup.Throttles {
		interval, err := time.ParseDuration(duration)
		if err != nil {
			log.WithField("name", name).WithField("timeWindow", duration).Warn("Failed to parse duration for throttle! Skipping throttle, this will invalidate any cycles which use this throttle.")
			continue
		}

		throttle, _ := NewThrottle(interval, 1) // TODO: do we need to cancel?
		scheduler.throttles[name] = throttle
	}

	for _, cycle := range setup.LongTermCycles {
		throttle, ok := scheduler.throttles[cycle.Throttle]
		if !ok {
			log.WithField("name", cycle.Name).WithField("throttle", cycle.Throttle).Warn("No throttle found for this cycle! Skipping this cycle.")
			continue
		}

		notifier := cms.NewNotifier()
		reader := native.NewMongoNativeReader(mongo, cycle.Collection)
		task := tasks.NewNativeContentPublishTask(reader, notifier)
		longTermCycle := NewLongTermCycle(cycle.Name, mongo, cycle.Collection, throttle, task)
		scheduler.Add(longTermCycle)
	}

	for _, cycle := range setup.ShortTermCycles {
		timeWindow, err := time.ParseDuration(cycle.TimeWindow)
		if err != nil {
			log.WithField("name", cycle.Name).WithField("timeWindow", cycle.TimeWindow).Warn("Failed to parse duration for short term cycle! Skipping cycle.")
			continue
		}

		notifier := cms.NewNotifier()
		reader := native.NewMongoNativeReader(mongo, cycle.Collection)
		task := tasks.NewNativeContentPublishTask(reader, notifier)
		shortTermCycle := NewShortTermCycle(cycle.Name, mongo, cycle.Collection, timeWindow, task)
		scheduler.Add(shortTermCycle)
	}

	return scheduler, nil
}
