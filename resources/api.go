package resources

import (
	"net/http"
	"strings"

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

		swagger["basePath"] = getBasePath(r)
		swagger["schemes"] = getSchemes(r)

		host, ok := getHost(r)
		if ok {
			swagger["host"] = host
		}

		updatedAPI, err := yaml.Marshal(swagger)
		if err != nil {
			outputStaticAPI(w, api)
			return
		}

		w.Header().Add("Content-Type", "text/vnd.yaml")
		w.Write(updatedAPI)
	}
}

func getHost(r *http.Request) (string, bool) {
	if r.Host == "" {
		return "api.ft.com", false
	}
	return r.Host, true
}

func getBasePath(r *http.Request) string {
	if r.URL.Path == "" || r.URL.Path == "/__api" || !strings.HasSuffix(r.URL.Path, "/__api") {
		return "/"
	}

	return strings.TrimSuffix(r.URL.Path, "/__api")
}

func getSchemes(r *http.Request) []string {
	if strings.TrimSpace(r.URL.Scheme) == "" {
		return []string{"http", "https"}
	}

	return []string{r.URL.Scheme}
}

func outputStaticAPI(w http.ResponseWriter, api []byte) {
	w.Header().Add("Content-Type", "text/vnd.yaml")
	w.Write(api)
}
