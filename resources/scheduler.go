package resources

import (
	"net/http"

	"github.com/Financial-Times/publish-carousel/scheduler"
	log "github.com/Sirupsen/logrus"
)

// ShutdownScheduler stops all cycles
func ShutdownScheduler(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := sched.Shutdown()
		if err != nil {
			log.WithError(err).Error("Error in shutting down the scheduler")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// StartScheduler resumes all cycles
func StartScheduler(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := sched.Start()
		if err != nil {
			log.WithError(err).Error("Error in starting the scheduler")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
	}
}
