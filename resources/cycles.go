package resources

import (
	"encoding/json"
	"net/http"

	"github.com/Financial-Times/publish-carousel/scheduler"
	log "github.com/Sirupsen/logrus"
)

func GetCycles(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		c := sched.Cycles()
		data, err := json.Marshal(c)
		if err != nil {
			log.WithError(err).Warn("Error in marshalling cycles")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(data)
	}
}

// CREATE POST on /cycles
func CreateCycle(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var cycleConfig scheduler.CycleConfig
		err := decoder.Decode(&cycleConfig)
		if err != nil {
			log.WithError(err).Info("Invalid JSON in the request body")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Add("Content-Type", "application/json")

		err = sched.AddCycle(cycleConfig)
		if err != nil {
			log.WithError(err).WithField("cycleName", cycleConfig.Name).Warn("Error creating the cycle")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusCreated)
	}
}

// UPDATE PUT on /cycles/<id>
// DELETE DELETE on /cycles/<id>
// Pause POST on /cycles/<id>/pause
// Resume POST on /cycles/<id>/resume
