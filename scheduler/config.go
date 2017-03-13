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
	Cycles    []CycleConfig     `yaml:"cycles"`
}

type CycleConfig struct {
	Name       string `yaml:"name" json:"name"`
	Type       string `yaml:"type" json:"type"`
	Collection string `yaml:"collection" json:"collection"`
	Throttle   string `yaml:"throttle" json:"throttle"`
	TimeWindow string `yaml:"timeWindow" json:"timeWindow"`
	CoolDown   string `yaml:"coolDown" json:"coolDown"`
}

// Validate checks the provided config for errors
func (c CycleConfig) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("Please provide a cycle name")
	}

	if strings.TrimSpace(c.Collection) == "" {
		return errors.New("Please provide a valid native collection")
	}

	switch strings.ToLower(c.Type) {
	case "throttledwholecollection":
		if strings.TrimSpace(c.Throttle) == "" {
			return fmt.Errorf("Please provide a valid throttle name for cycle %v", c.Name)
		}

	case "fixedwindow":
		if _, err := time.ParseDuration(c.TimeWindow); err != nil {
			return fmt.Errorf("Error in parsing time window for cycle %v: %v", c.Name, err)
		}

	case "scalingwindow":
		if _, err := time.ParseDuration(c.TimeWindow); err != nil {
			return fmt.Errorf("Error in parsing time window for cycle %v: %v", c.Name, err)
		} else if _, err := time.ParseDuration(c.CoolDown); err != nil {
			return fmt.Errorf("Error in parsing cool down for cycle %v: %v", c.Name, err)
		}
	default:
		return fmt.Errorf("Please provide a valid type for cycle %v", c.Name)
	}

	return nil
}

// LoadSchedulerFromFile loads cycles and throttles from the provided yaml config file
func LoadSchedulerFromFile(configFile string, mongo native.DB, publishTask tasks.Task, rw MetadataReadWriter) (Scheduler, error) {
	scheduler := NewScheduler(mongo, publishTask, rw)

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
