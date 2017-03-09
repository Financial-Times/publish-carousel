package resources

import (
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/s3"
)

// Health returns a handler for the standard FT healthchecks
func Health(db native.DB, s3Service s3.ReadWriter) func(w http.ResponseWriter, r *http.Request) {
	return fthealth.Handler("publish-carousel", "A microservice that continuously republishes content and annotations available in the native store.", getHealthchecks(db, s3Service)...)
}

// GTG returns a handler for a standard GTG endpoint.
func GTG(db native.DB, s3Service s3.ReadWriter) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := []func() (string, error){pingMongo(db), pingS3(s3Service)}

		for _, check := range checks {
			_, err := check()
			if err != nil {
				w.WriteHeader(500)
				return
			}
		}

		w.WriteHeader(200)
	}
}

func getHealthchecks(db native.DB, s3Service s3.ReadWriter) []fthealth.Check {
	return []fthealth.Check{
		{
			Name:             "CheckConnectivityToNativeDatabase",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "The service is unable to connect to MongoDB. Content will not be periodically republished.",
			Severity:         1,
			PanicGuide:       "https://dewey.ft.com/upp-publish-carousel.html",
			Checker:          pingMongo(db),
		},
		{
			Name:             "CheckConnectivityToS3",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "The service is unable to connect to S3, which prevents the reading and writing of Carousel cycle state information, which will force the carousel to restart all cycles from the beginning.",
			Severity:         1,
			PanicGuide:       "https://dewey.ft.com/upp-publish-carousel.html",
			Checker:          pingS3(s3Service),
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

func pingS3(svc s3.ReadWriter) func() (string, error) {
	return func() (string, error) {
		return "", svc.Ping()
	}
}
