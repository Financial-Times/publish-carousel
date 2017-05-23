package resources

import (
	"net/http"
	"net/http/httptest"

	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/husobee/vestigo"
)

func setupRouter(sched scheduler.Scheduler, req *http.Request) *httptest.ResponseRecorder {
	r := vestigo.NewRouter()
	r.Get("/cycles", GetCycles(sched))
	r.Post("/cycles", CreateCycle(sched))

	r.Get("/cycles/:id", GetCycleForID(sched))
	r.Delete("/cycles/:id", DeleteCycle(sched))

	r.Get("/cycles/:id/throttle", GetCycleThrottle(sched))
	r.Put("/cycles/:id/throttle", SetCycleThrottle(sched))

	r.Post("/cycles/:id/resume", ResumeCycle(sched))

	r.Post("/cycles/:id/stop", StopCycle(sched))

	r.Post("/cycles/:id/reset", ResetCycle(sched))

	r.Post("/scheduler/start", StartScheduler(sched))

	r.Post("/scheduler/shutdown", ShutdownScheduler(sched))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
