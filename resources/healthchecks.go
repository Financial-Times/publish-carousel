package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/sirupsen/logrus"
)

type HealthService struct {
	healthCheck fthealth.HealthCheck
}

func NewHealthService(appSystemCode string, appName string, description string, db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error, upServices ...cluster.Service) *HealthService {
	service := &HealthService{
		healthCheck: fthealth.HealthCheck{
			SystemCode:  appSystemCode,
			Name:        appName,
			Description: description,
		},
	}
	service.healthCheck.Checks = service.getHealthchecks(db, s3Service, notifier, sched, configError, upServices...)
	return service
}

// Health returns a handler for the standard FT healthchecks
func (healthService *HealthService) Health() func(w http.ResponseWriter, r *http.Request) {
	hc := fthealth.TimedHealthCheck{
		HealthCheck: healthService.healthCheck,
		Timeout:     10 * time.Second,
	}
	return fthealth.Handler(hc)
}

func (healthService *HealthService) gtgCheck(check fthealth.Check) gtg.Status {
	if _, err := check.Checker(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

// GTG returns a handler for a standard GTG endpoint.
func (healthService *HealthService) GTG() gtg.Status {
	var checks []gtg.StatusChecker

	for _, check := range healthService.healthCheck.Checks {
		status := healthService.gtgCheck(check)
		checks = append(checks, func() gtg.Status {
			return status
		})
	}

	return gtg.FailFastParallelCheck(checks)()
}

func (healthService *HealthService) getHealthchecks(db native.DB, s3Service s3.ReadWriter, notifier cms.Notifier, sched scheduler.Scheduler, configError error, upServices ...cluster.Service) []fthealth.Check {
	return []fthealth.Check{
		{
			Name:             "CheckConnectivityToNativeDatabase",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "The service is unable to connect to MongoDB. Content will not be periodically republished.",
			Severity:         1,
			PanicGuide:       "https://runbooks.in.ft.com/publish-carousel",
			Checker:          pingMongo(db),
		},
		{
			Name:             "CheckConnectivityToS3",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "The service is unable to connect to S3, which prevents the reading and writing of Carousel cycle state information, which will force the carousel to restart all cycles from the beginning.",
			Severity:         1,
			PanicGuide:       "https://runbooks.in.ft.com/publish-carousel",
			Checker:          pingS3(s3Service),
		},
		{
			Name:             "CheckCMSNotifierHealth",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "The CMS Notifier service is unhealthy. Carousel publishes may fail, and will not be retried until the next cycle. ",
			Severity:         1,
			PanicGuide:       "https://runbooks.in.ft.com/publish-carousel",
			Checker:          cmsNotifierGTG(notifier),
		},
		{
			Name:             "UnhealthyCycles",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: "At least one of the Carousel cycles is unhealthy. This should be investigated.",
			Severity:         1,
			PanicGuide:       "https://runbooks.in.ft.com/publish-carousel",
			Checker:          unhealthyCycles(sched),
		},
		{
			Name:             "InvalidCycleConfiguration",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: `At least one error occurred while intialising cycles from the "cycles.yml" file.`,
			Severity:         1,
			PanicGuide:       "https://runbooks.in.ft.com/publish-carousel",
			Checker:          configHealthcheck(configError),
		},
		{
			Name:             "UnhealthyCluster",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: `If the cluster is unhealthy, the Carousel scheduler will shutdown until the system has stabilised.`,
			Severity:         1,
			PanicGuide:       "https://runbooks.in.ft.com/publish-carousel",
			Checker:          unhealthyClusters(sched, upServices...),
		},
		{
			Name:             "ActivePublishingCluster",
			BusinessImpact:   "No Business Impact.",
			TechnicalSummary: `In case of a failover of the publishing cluster, the Carousel will be automatically disabled.`,
			Severity:         1,
			PanicGuide:       "https://runbooks.in.ft.com/publish-carousel",
			Checker:          clusterFailoverHealthcheck(sched),
		},
	}
}

func pingMongo(db native.DB) func() (string, error) {
	return func() (string, error) {
		tx, err := db.Open()
		if err != nil {
			return "", err
		}

		defer func() { go tx.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = tx.Ping(ctx)
		if err != nil {
			return "", err
		}

		return "OK", nil
	}
}

func pingS3(svc s3.ReadWriter) func() (string, error) {
	return func() (string, error) {
		err := svc.Ping()
		if err != nil {
			return "", err
		}

		return "OK", nil
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

		return "No unhealthy cycles.", nil
	}
}

func cmsNotifierGTG(notifier cms.Notifier) func() (string, error) {
	return func() (string, error) {
		err := notifier.Check()
		if err != nil {
			return "", err
		}

		return "OK", nil
	}
}

func toJSON(data interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}

type checkResult struct {
	serviceName string
	err         error
}

func unhealthyClusters(sched scheduler.Scheduler, upServices ...cluster.Service) func() (string, error) {
	return func() (string, error) {
		results := make(chan checkResult, 1)

		checkServicesGTG(upServices, results)
		unhealthyServices, msg := getUnhealthyServices(len(upServices), results)

		if len(*unhealthyServices) > 0 {
			if sched.IsRunning() {
				log.WithFields(log.Fields{"services": *unhealthyServices}).Info("Shutting down scheduler due to unhealthy cluster service(s)")
				sched.Shutdown()
			}
			return fmt.Sprintf("One or more dependent services are unhealthy: %v", toJSON(*unhealthyServices)), errors.New(msg)
		}

		if !sched.IsRunning() && sched.IsEnabled() && !sched.WasAutomaticallyDisabled() {
			log.Info("Cluster health back to normal; restarting scheduler.")
			sched.Start()
		}

		return "Cluster is healthy", nil
	}
}

func checkServicesGTG(upServices []cluster.Service, results chan<- checkResult) {
	for _, service := range upServices {
		go func(svc cluster.Service) {
			err := svc.Check()
			if err != nil {
				results <- checkResult{serviceName: svc.Name(), err: err}
			} else {
				results <- checkResult{}
			}
		}(service)
	}
}

func getUnhealthyServices(serviceCount int, results chan checkResult) (*[]string, string) {
	var unhealthyServices []string
	var errMsg string
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var checkedServices int
		for result := range results {
			checkedServices++
			if result.serviceName != "" {
				unhealthyServices = append(unhealthyServices, result.serviceName)
				errMsg += result.err.Error() + ". "
			}
			if checkedServices == serviceCount {
				close(results)
			}
		}
	}()
	wg.Wait()
	return &unhealthyServices, errMsg
}

func configHealthcheck(err error) func() (string, error) {
	return func() (string, error) {
		if err != nil {
			return "", err
		}

		return "OK", nil
	}
}

func clusterFailoverHealthcheck(s scheduler.Scheduler) func() (string, error) {
	return func() (string, error) {
		if s.IsAutomaticallyDisabled() {
			return "Detected publishing cluster failover", errors.New("carousel scheduler has been automatically disabled")
		}
		if s.WasAutomaticallyDisabled() {
			return "Detected publishing cluster failback", errors.New("carousel scheduler is enabled but stopped")
		}
		if !s.IsEnabled() {
			return "Carousel scheduler manually disabled", nil
		}
		return "No failover issues, carousel scheduler enabled", nil
	}
}
