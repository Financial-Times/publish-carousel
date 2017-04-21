package resources

import "net/http"

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
