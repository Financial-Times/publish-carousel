package resources

import (
	"encoding/json"
	"net/http"

	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/Sirupsen/logrus"
)

func GetCycles(sched scheduler.Scheduler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		c := sched.Cycles()
		data, err := json.Marshal(c)
		if err != nil {
			logrus.WithError(err).Error("oh no")
		}

		w.Write(data)
	}
}
