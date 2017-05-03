package resources

import (
	"net/http"
	"net/url"

	yaml "gopkg.in/yaml.v2"
)

// API returns the swagger.yml for this service.
func API(api []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(api) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		swagger := make(map[string]interface{})
		err := yaml.Unmarshal(api, swagger)
		if err != nil {
			outputStaticAPI(w, api)
			return
		}

		uri, _ := url.Parse(r.RequestURI) // must be a valid url

		swagger["host"] = uri.Host
		updatedAPI, err := yaml.Marshal(swagger)
		if err != nil {
			outputStaticAPI(w, api)
			return
		}

		w.Header().Add("Content-Type", "text/vnd.yaml")
		w.Write(updatedAPI)
	}
}

func outputStaticAPI(w http.ResponseWriter, api []byte) {
	w.Header().Add("Content-Type", "text/vnd.yaml")
	w.Write(api)
}
