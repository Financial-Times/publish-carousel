package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/scheduler"
	log "github.com/Sirupsen/logrus"
)

// Health returns a handler for the standard FT healthchecks
func Health(db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error, upServices ...cluster.Service) func(w http.ResponseWriter, r *http.Request) {
	return fthealth.Handler("publish-carousel", "A microservice that continuously republishes content and annotations available in the native store.", getHealthchecks(db, s3Service, notifier, sched, configError, upServices...)...)
}

// GTG returns a handler for a standard GTG endpoint.
func GTG(db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error, upServices ...cluster.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := []func() (string, error){pingMongo(db), pingS3(s3Service), cmsNotifierGTG(notifier), unhealthyCycles(sched), configHealthcheck(configError), unhealthyClusters(sched, upServices...)}

		for _, check := range checks {
			_, err := check()
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func getHealthchecks(db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error, upServices ...cluster.Service) []fthealth.Check {
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
		{
			Name:             "UnhealthyCluster",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: `If the cluster is unhealthy, the Carousel scheduler will shutdown until the system has stabilised.`,
			Severity:         1,
			PanicGuide:       "https://dewey.ft.com/upp-publish-carousel.html",
			Checker:          unhealthyClusters(sched, upServices...),
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
			return "", errors.New("The following cycles are unhealthy! " + toJSON(unhealthyIDs))
		}

		return "", nil
	}
}

func cmsNotifierGTG(notifier cms.Notifier) func() (string, error) {
	return func() (string, error) {
		return "", notifier.GTG()
	}
}

func toJSON(data interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}

func unhealthyClusters(sched scheduler.Scheduler, upServices ...cluster.Service) func() (string, error) {
	return func() (string, error) {
		var unhealthyServices []string

		errs := make([]error, 0)
		for _, service := range upServices {
			err := service.GTG()
			if err != nil {
				if sched.IsRunning() {
					log.WithField("service", service.Name()).Info("Shutting down scheduler due to unhealthy cluster service(s)")
					sched.Shutdown()
				}
				unhealthyServices = append(unhealthyServices, service.Name()+": "+service.URL())
				errs = append(errs, err)
			}
		}

		msg := ""
		for _, err := range errs {
			msg += err.Error() + ". "
		}

		if len(unhealthyServices) > 0 {
			return fmt.Sprintf("One or more dependent services are unhealthy: %v", toJSON(unhealthyServices)), errors.New(msg)
		}

		if !sched.IsRunning() && sched.IsEnabled() {
			log.Info("Cluster health back to normal; restarting scheduler.")
			sched.Start()
		}

		return "Cluster is healthy", nil
	}
}

func configHealthcheck(err error) func() (string, error) {
	return func() (string, error) {
		return "", err
	}
}
