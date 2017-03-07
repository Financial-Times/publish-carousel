package scheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

type cycleSetupConfig struct {
	Throttles map[string]string `yaml:"throttles"`
	Cycles    []cycleConfig     `yaml:"cycles"`
}

type cycleConfig struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Collection string `yaml:"collection"`
	Throttle   string `yaml:"throttle"`
	TimeWindow string `yaml:"timeWindow"`
}

func (c *cycleConfig) validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("Please provide a cycle name")
	}
	if strings.TrimSpace(c.Collection) == "" {
		return errors.New("Please provide a valid native collection")
	}
	switch strings.ToLower(c.Type) {
	case "longterm":
		if strings.TrimSpace(c.Throttle) == "" {
			return fmt.Errorf("Please provide a valid throttle name for cycle %v", c.Name)
		}
	case "shortterm":
		if _, err := time.ParseDuration(c.TimeWindow); err != nil {
			return fmt.Errorf("Error in parsing duration for cycle %v: %v", c.Name, err)
		}
	default:
		return fmt.Errorf("Please provide a valid type for cycle %v", c.Name)
	}
	return nil
}

func LoadSchedulerFromFile(configFile string, mongo native.DB, publishTask tasks.Task) (Scheduler, error) {
	scheduler := NewScheduler(mongo, publishTask)

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
		err := scheduler.AddThrottle(name, duration)
		if err != nil {
			log.WithError(err).WithField("throttleName", name).WithField("timeWindow", duration).Warn("Skipping throttle, this will invalidate any cycles which use this throttle.")
		}
	}

	for _, cycle := range setup.Cycles {
		err := scheduler.AddCycle(cycle)
		if err != nil {
			log.WithError(err).WithField("cycleName", cycle.Name).Warn("Skipping cycle")
		}
	}

	return scheduler, nil
}
