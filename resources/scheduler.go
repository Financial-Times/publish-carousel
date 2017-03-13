package resources

import (
	"net/http"

	"github.com/Financial-Times/publish-carousel/scheduler"
)

// ShutdownScheduler stops all cycles
func ShutdownScheduler(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sched.Shutdown()
		w.WriteHeader(http.StatusOK)
	}
}

// StartScheduler resumes all cycles
func StartScheduler(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sched.Start()
		w.WriteHeader(http.StatusOK)
	}
}
