package resources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// API returns the swagger.yml for this service.
func API(api []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(api) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.Header().Add("Content-Type", "text/vnd.yaml")
		w.Write(api)
	}
}

type logLevelUpdate struct {
	Level string `json:"level"`
}

// LogLevel handles request to change the logging level between debug and info
func LogLevel(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	update := logLevelUpdate{}
	err := dec.Decode(&update)

	if err != nil {
		http.Error(w, "Failed to parse log level update request", http.StatusBadRequest)
		return
	}

	lowLevel := strings.ToLower(update.Level)
	switch lowLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	default:
		http.Error(w, `Invalid level. Please select one of "debug" or "info"`, http.StatusBadRequest)
		return
	}

	w.Write([]byte(fmt.Sprintf(`Updated log level to "%v"`, lowLevel)))
	w.WriteHeader(http.StatusOK)
}
