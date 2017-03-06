package resources

import (
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/publish-carousel/native"
)

// Health returns a handler for the standard FT healthchecks
func Health(db native.DB) func(w http.ResponseWriter, r *http.Request) {
	return fthealth.Handler("publish-carousel", "A microservice that continuously republishes content and annotations available in the native store.", getHealthchecks(db)[0])
}

// GTG returns a handler for a standard GTG endpoint.
func GTG(db native.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := pingMongo(db)()
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
	}
}

func getHealthchecks(db native.DB) []fthealth.Check {
	return []fthealth.Check{
		{
			Name:             "CheckConnectivityToNativeDatabase",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "The service is unable to connect to MongoDB. Content will not be periodically republished.",
			Severity:         1,
			PanicGuide:       "https://dewey.ft.com/upp-publish-carousel.html",
			Checker:          pingMongo(db),
		},
	}
}

func pingMongo(db native.DB) func() (string, error) {
	return func() (string, error) {
		tx, err := db.Open()
		if err != nil {
			return "", err
		}

		defer tx.Close()

		return "", tx.Ping()
	}
}
