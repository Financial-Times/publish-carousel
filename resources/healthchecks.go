package resources

import (
	"encoding/json"
	"errors"
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/scheduler"
)

// Health returns a handler for the standard FT healthchecks
func Health(db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error) func(w http.ResponseWriter, r *http.Request) {
	return fthealth.Handler("publish-carousel", "A microservice that continuously republishes content and annotations available in the native store.", getHealthchecks(db, s3Service, notifier, sched, configError)...)
}

// GTG returns a handler for a standard GTG endpoint.
func GTG(db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := []func() (string, error){pingMongo(db), pingS3(s3Service), cmsNotifierGTG(notifier), unhealthyCycles(sched), configHealthcheck(configError)}

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

func getHealthchecks(db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error) []fthealth.Check {
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
		{
			Name:             "CheckCMSNotifierHealth",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "The CMS Notifier service is unhealthy. Carousel publishes may fail, and will not be retried until the next cycle. ",
			Severity:         1,
			PanicGuide:       "https://dewey.ft.com/upp-publish-carousel.html",
			Checker:          cmsNotifierGTG(notifier),
		},
		{
			Name:             "UnhealthyCycles",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "At least one of the Carousel cycles is unhealthy. This should be investigated.",
			Severity:         1,
			PanicGuide:       "https://dewey.ft.com/upp-publish-carousel.html",
			Checker:          unhealthyCycles(sched),
		},
		{
			Name:             "InvalidCycleConfiguration",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: `At least one error occurred while intialising cycles from the "cycles.yml" file.`,
			Severity:         1,
			PanicGuide:       "https://dewey.ft.com/upp-publish-carousel.html",
			Checker:          configHealthcheck(configError),
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

func unhealthyCycles(sched scheduler.Scheduler) func() (string, error) {
	return func() (string, error) {
		var unhealthyIDs []string
		for _, cycle := range sched.Cycles() {
			for _, state := range cycle.Metadata().State {
				if state == "unhealthy" {
					unhealthyIDs = append(unhealthyIDs, cycle.ID())
				}
			}
		}

		if len(unhealthyIDs) > 0 {
			j, _ := json.Marshal(unhealthyIDs)
			return "", errors.New("The following cycles are unhealthy! " + string(j))
		}

		return "", nil
	}
}

func cmsNotifierGTG(notifier cms.Notifier) func() (string, error) {
	return func() (string, error) {
		return "", notifier.GTG()
	}
}

func configHealthcheck(err error) func() (string, error) {
	return func() (string, error) {
		return "", err
	}
}
