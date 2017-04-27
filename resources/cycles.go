package resources

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Financial-Times/publish-carousel/scheduler"
	log "github.com/Sirupsen/logrus"
	"github.com/husobee/vestigo"
)

// GetCycles returns all cycles as an array
func GetCycles(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		cycles := sched.Cycles()

		arr := make([]scheduler.Cycle, 0)
		for _, c := range cycles {
			arr = append(arr, c)
		}

		data, err := json.Marshal(arr)
		if err != nil {
			log.WithError(err).Warn("Error in marshalling cycles")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(data)
	}
}

// GetCycleForID returns the individual cycle
func GetCycleForID(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		cycle, err := findCycle(sched, w, r)
		if err != nil {
			return
		}

		data, err := json.Marshal(cycle)
		if err != nil {
			log.WithError(err).Info("Failed to marshal cycles.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(data)
	}
}

// CreateCycle POST request to create a new cycle
func CreateCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var cycleConfig scheduler.CycleConfig

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&cycleConfig)

		if err != nil {
			log.Warn("failed to decode body")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = cycleConfig.Validate()
		if err != nil {
			log.Warn("failed to validate cycle")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cycle, err := createCycle(sched, &cycleConfig, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Location", cycleURL(cycle))
	}
}

// DeleteCycle deletes the cycle by the given id
func DeleteCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := vestigo.Param(r, "id")
		err := sched.DeleteCycle(id)

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ResumeCycle resumes the stopped cycle.
func ResumeCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cycles := sched.Cycles()

		cycle, ok := cycles[vestigo.Param(r, "id")]
		if !ok {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		//TODO: Add stopped validation?
		cycle.Start()
		w.WriteHeader(http.StatusOK)
	}
}

// ResetCycle stops and completely resets the given cycle
func ResetCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cycles := sched.Cycles()

		cycle, ok := cycles[vestigo.Param(r, "id")]
		if !ok {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		cycle.Reset()
		w.WriteHeader(http.StatusOK)
	}
}

// StopCycle stops the given cycle ID
func StopCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cycles := sched.Cycles()

		cycle, ok := cycles[vestigo.Param(r, "id")]
		if !ok {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		cycle.Stop()
		w.WriteHeader(http.StatusOK)
	}
}

// Get a cycle throttle
func GetCycleThrottle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		cycle, err := findCycle(sched, w, r)
		if err != nil {
			return
		}

		switch cycle.Type() {
		case scheduler.ThrottledWholeCollectionType:
			throttledCycle, ok := cycle.(*scheduler.ThrottledWholeCollectionCycle)
			if !ok {
				log.WithError(err).Info("Failed to cast cycle.")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			data, err := json.Marshal(throttledCycle.Throttle)
			if err != nil {
				log.WithError(err).Info("Failed to marshal cycle throttle.")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Write(data)

		default:
			log.WithField("cycleID", cycle.ID()).Info("cycle is not throttled")
			http.Error(w, fmt.Sprintf("Cycle is not throttled: %v", cycle.ID()), http.StatusNotFound)
		}
	}
}

// Set a cycle throttle
func SetCycleThrottle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		cycle, err := findCycle(sched, w, r)
		if err != nil {
			return
		}

		cycleID := cycle.ID()
		throttledCycle, ok := cycle.(*scheduler.ThrottledWholeCollectionCycle)
		if !ok {
			log.WithField("cycleID", cycleID).Info("cycle is not throttled")
			http.Error(w, fmt.Sprintf("Cycle is not throttled: %v", cycleID), http.StatusBadRequest)
			return
		}

		var newThrottle scheduler.DefaultThrottle

		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&newThrottle)
		log.Infof("new throttle = %v", newThrottle)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		metadata := cycle.Metadata()

		sched.DeleteCycle(cycleID)

		config := throttledCycle.TransformToConfig()
		config.Throttle = newThrottle.Interval().String()

		newCycle, err := createCycle(sched, config, &metadata)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, cycleURL(newCycle), http.StatusSeeOther)
	}
}

func findCycle(sched scheduler.Scheduler, w http.ResponseWriter, r *http.Request) (scheduler.Cycle, error) {
	cycles := sched.Cycles()
	cycleID := vestigo.Param(r, "id")
	cycle, ok := cycles[cycleID]
	if !ok {
		log.WithField("cycleID", cycleID).Warn("Cycle not found")
		err := fmt.Errorf("Cycle not found with ID: %v", cycleID)
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, err
	}

	return cycle, nil
}

func createCycle(sched scheduler.Scheduler, cycleConfig *scheduler.CycleConfig, metadata *scheduler.CycleMetadata) (scheduler.Cycle, error) {
	cycle, err := sched.NewCycle(*cycleConfig)
	if err != nil {
		log.WithError(err).WithField("cycle", cycleConfig.Name).Warn("Failed to create new cycle.")
		return nil, err
	}
	log.Infof("new cycle = %v", cycle)

	if metadata != nil {
		cycle.SetMetadata(*metadata)
	}

	err = sched.AddCycle(cycle)
	if err != nil {
		log.WithError(err).WithField("cycle", cycleConfig.Name).Warn("Failed to add the cycle to the scheduler")
		return nil, err
	}

	return cycle, nil
}

func cycleURL(cycle scheduler.Cycle) string {
	return fmt.Sprintf("/cycles/%v", cycle.ID())
}
