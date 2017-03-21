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
	Name            string `yaml:"name" json:"name"`
	Type            string `yaml:"type" json:"type"`
	Origin          string `yaml:"origin" json:"origin"`
	Collection      string `yaml:"collection" json:"collection"`
	CoolDown        string `yaml:"coolDown" json:"coolDown"`
	Throttle        string `yaml:"throttle" json:"throttle,omitempty"`
	TimeWindow      string `yaml:"timeWindow" json:"timeWindow,omitempty"`
	MinimumThrottle string `yaml:"minimumThrottle" json:"minimumThrottle,omitempty"`
	MaximumThrottle string `yaml:"maximumThrottle" json:"maximumThrottle,omitempty"`
}

// Validate checks the provided config for errors
func (c CycleConfig) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("Please provide a cycle name")
	}

	if strings.TrimSpace(c.Collection) == "" {
		return errors.New("Please provide a valid native collection")
	}

	if strings.TrimSpace(c.Origin) == "" {
		return errors.New("Please provide a valid X-Origin-System-Id")
	}

	if err := checkDurations(c.CoolDown); err != nil {
		return err
	}

	switch strings.ToLower(c.Type) {
	case "throttledwholecollection":
		if strings.TrimSpace(c.Throttle) == "" {
			return fmt.Errorf("Please provide a valid throttle name for cycle %v", c.Name)
		}

	case "fixedwindow":
		if err := checkDurations(c.Name, c.TimeWindow, c.MinimumThrottle); err != nil {
			return err
		}
	case "scalingwindow":
		if err := checkDurations(c.Name, c.TimeWindow, c.MinimumThrottle, c.MaximumThrottle); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Please provide a valid type for cycle %v", c.Name)
	}

	return nil
}

func checkDurations(name string, durations ...string) error {
	for _, duration := range durations {
		if _, err := time.ParseDuration(duration); err != nil {
			return fmt.Errorf("Error in parsing duration for cycle %v: Duration=%v err=%v.", name, duration, err)
		}
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
			log.WithError(err).WithField("throttle", name).WithField("timeWindow", duration).Warn("Skipping throttle, this will invalidate any cycles which use this throttle.")
		}
	}

	for _, cycle := range setup.Cycles {
		err := scheduler.AddCycle(cycle)
		if err != nil {
			log.WithError(err).WithField("cycle", cycle.Name).Warn("Skipping cycle")
		}
	}

	return scheduler, nil
}
