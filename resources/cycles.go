package resources

import (
	"encoding/json"
	"net/http"

	"github.com/Financial-Times/publish-carousel/scheduler"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// GetCycles returns all cycles as an array
func GetCycles(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		cycles := sched.Cycles()

		var arr []scheduler.Cycle
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
		cycles := sched.Cycles()

		vars := mux.Vars(r)
		cycle, ok := cycles[vars["id"]]
		if !ok {
			http.Error(w, "", http.StatusNotFound)
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = cycleConfig.Validate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = sched.AddCycle(cycleConfig)
		if err != nil {
			log.WithError(err).WithField("cycle", cycleConfig.Name).Warn("Failed to create new cycle.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

// DeleteCycle deletes the cycle by the given id
func DeleteCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		err := sched.DeleteCycle(id)

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func ResumeCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cycles := sched.Cycles()

		vars := mux.Vars(r)
		cycle, ok := cycles[vars["id"]]
		if !ok {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		cycle.Start()
		w.WriteHeader(http.StatusOK)
	}
}

// ResetCycle stops and completely resets the given cycle
func ResetCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cycles := sched.Cycles()

		vars := mux.Vars(r)
		cycle, ok := cycles[vars["id"]]
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

		vars := mux.Vars(r)
		cycle, ok := cycles[vars["id"]]
		if !ok {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		cycle.Stop()
		w.WriteHeader(http.StatusOK)
	}
}

// UPDATE PUT on /cycles/<id>
