package resources

import (
	"net/http"
	"net/http/httptest"

	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/gorilla/mux"
)

func setupRouter(sched scheduler.Scheduler, req *http.Request) *httptest.ResponseRecorder {
	r := mux.NewRouter()
	r.HandleFunc("/cycles", GetCycles(sched)).Methods("GET")
	r.HandleFunc("/cycles", CreateCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles", MethodNotAllowed()).Methods("PUT", "DELETE")

	r.HandleFunc("/cycles/{id}", GetCycleForID(sched)).Methods("GET")
	r.HandleFunc("/cycles/{id}", DeleteCycle(sched)).Methods("DELETE")
	r.HandleFunc("/cycles/{id}", MethodNotAllowed()).Methods("PUT", "POST")

	r.HandleFunc("/cycles/{id}/throttle", GetCycleThrottle(sched)).Methods("GET")
	r.HandleFunc("/cycles/{id}/throttle", SetCycleThrottle(sched)).Methods("PUT")

	r.HandleFunc("/cycles/{id}/resume", ResumeCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/resume", MethodNotAllowed()).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/cycles/{id}/stop", StopCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/stop", MethodNotAllowed()).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/cycles/{id}/reset", ResetCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/reset", MethodNotAllowed()).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/scheduler/start", StartScheduler(sched)).Methods("POST")
	r.HandleFunc("/scheduler/start", MethodNotAllowed()).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/scheduler/shutdown", ShutdownScheduler(sched)).Methods("POST")
	r.HandleFunc("/scheduler/shutdown", MethodNotAllowed()).Methods("GET", "PUT", "DELETE")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
