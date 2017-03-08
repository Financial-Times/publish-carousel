package scheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/tasks"
	log "github.com/Sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

type cycleSetupConfig struct {
	Throttles map[string]string `yaml:"throttles"`
	Cycles    []CycleConfig     `yaml:"cycles"`
}

type CycleConfig struct {
	Name       string `yaml:"name" json:"name"`
	Type       string `yaml:"type" json:"type"`
	Collection string `yaml:"collection" json:"collection"`
	Throttle   string `yaml:"throttle" json:"throttle"`
	TimeWindow string `yaml:"timeWindow" json:"timeWindow"`
}

func (c *CycleConfig) validate() error {
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

func LoadSchedulerFromFile(configFile string, mongo native.DB, publishTask tasks.Task, s3RW s3.S3ReadWrite) (Scheduler, error) {
	scheduler := NewScheduler(mongo, publishTask, s3RW)

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
