package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ui "github.com/Financial-Times/publish-carousel-ui"
	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/etcd"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/resources"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	cli "gopkg.in/urfave/cli.v1"
)

func init() {
	f := &log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}

	log.SetFormatter(f)
}

func main() {
	app := cli.NewApp()
	app.Name = "publish-carousel"
	app.Usage = "A microservice that continuously republishes content and annotations available in the native store."

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "cycles",
			Value:  "./cycles.yml",
			EnvVar: "CYCLES_FILE",
			Usage:  "Path to the YML cycle configuration file.",
		},
		cli.StringFlag{
			Name:   "blacklist",
			Value:  "./carousel_blacklist.txt",
			EnvVar: "BLACKLIST_FILE",
			Usage:  "Path to the plaintxt blacklist file, which contains blacklisted uuids.",
		},
		cli.StringFlag{
			Name:   "mongo-db",
			Value:  "localhost:27017",
			EnvVar: "MONGO_DB_URL",
			Usage:  "The Mongo DB connection url string (comma delimited).",
		},
		cli.StringFlag{
			Name:   "cms-notifier-url",
			Value:  "http://localhost:8080/__cms-notifier",
			EnvVar: "CMS_NOTIFIER_URL",
			Usage:  "The CMS Notifier instance to POST publishes to.",
		},
		cli.StringFlag{
			Name:   "pam-url",
			Value:  "http://localhost:8080/__publish-availability-monitor",
			EnvVar: "PAM_URL",
			Usage:  "The URL of the publish availability monitor to check the health of the cluster.",
		},
		cli.StringFlag{
			Name:   "lagcheck-url",
			Value:  "http://localhost:8080/__kafka-lagcheck",
			EnvVar: "LAGCHECK_URL",
			Usage:  "The URL of the queue lagcheck service to verify the health of the cluster.",
		},
		cli.StringFlag{
			Name:   "aws-region",
			EnvVar: "AWS_REGION",
			Value:  "",
			Usage:  "The AWS Region for this cluster.",
		},
		cli.StringFlag{
			Name:   "s3-bucket",
			EnvVar: "S3_BUCKET",
			Value:  "",
			Usage:  "The S3 Bucket to save carousel states.",
		},
		cli.IntFlag{
			Name:   "mongo-timeout",
			Value:  10000,
			EnvVar: "MONGO_DB_TIMEOUT",
			Usage:  "The timeout (in milliseconds) for Mongo DB connections.",
		},
		cli.StringSliceFlag{
			Name:   "etcd-peers",
			Value:  &cli.StringSlice{"http://localhost:2379"},
			EnvVar: "ETCD_PEERS",
			Usage:  `The list of ETCD peers (e,g. "http://localhost:2379")`,
		},
		cli.StringFlag{
			Name:   "toggle-etcd-key",
			Value:  "/ft/config/publish-carousel/enable",
			EnvVar: "TOGGLE_ETCD_KEY",
			Usage:  "The ETCD key that enables or disables the carousel",
		},
		cli.StringFlag{
			Name:   "default-throttle",
			Value:  "1m",
			EnvVar: "DEFAULT_THROTTLE",
			Usage:  "Default throttle for whole collection cycles, if it is not specified in configuration file",
		},
	}

	app.Action = func(ctx *cli.Context) {
		log.Info("Starting the Publish Carousel.")

		s3rw := s3.NewReadWriter(ctx.String("aws-region"), ctx.String("s3-bucket"))
		stateRw := scheduler.NewS3MetadataReadWriter(s3rw)

		blist, err := blacklist.NewBuilder().FilterImages().FileBasedBlacklist(ctx.String("blacklist")).Build()
		if err != nil {
			panic(err)
		}

		mongo := native.NewMongoDatabase(ctx.String("mongo-db"), ctx.Int("mongo-timeout"))

		reader := native.NewMongoNativeReader(mongo)
		notifier, err := cms.NewNotifier(ctx.String("cms-notifier-url"), &http.Client{Timeout: time.Second * 30})
		if err != nil {
			log.WithError(err).Error("Error in CMS Notifier configuration")
		}

		pam, err := cluster.NewService("publish-availability-monitor", ctx.String("pam-url"))
		if err != nil {
			log.WithError(err).Error("Error in Publish Availability Monitor configuration")
		}

		queueLagcheck, err := cluster.NewService("kafka-lagcheck", ctx.String("lagcheck-url"))
		if err != nil {
			log.WithError(err).Error("Error in Kafka lagcheck configuration")
		}

		task := tasks.NewNativeContentPublishTask(reader, notifier, blist)

		etcdWatcher, err := etcd.NewEtcdWatcher(ctx.StringSlice("etcd-peers"))

		if err != nil {
			panic(err)
		}

		defaultThrottle, err := time.ParseDuration(ctx.String("default-throttle"))

		if err != nil {
			log.WithError(err).Error("Invalid value for default throttle")
		}

		sched, configError := scheduler.LoadSchedulerFromFile(ctx.String("cycles"), mongo, task, stateRw, defaultThrottle)
		if err != nil {
			log.WithError(configError).Error("Failed to load cycles configuration file")
		}

		toggle, err := etcdWatcher.Read(ctx.String("toggle-etcd-key"))
		if err != nil {
			panic(err)
		}

		sched.ToggleHandler(toggle)

		go etcdWatcher.Watch(context.Background(), ctx.String("toggle-etcd-key"), sched.ToggleHandler)

		sched.RestorePreviousState()
		sched.Start()

		shutdown(sched)
		serve(mongo, sched, s3rw, notifier, configError, pam, queueLagcheck)
	}

	app.Run(os.Args)
}

func shutdown(sched scheduler.Scheduler) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		log.Info("Saving current carousel state to S3.")
		sched.SaveCycleMetadata()
		os.Exit(0)
	}()
}

func serve(mongo native.DB, sched scheduler.Scheduler, s3rw s3.ReadWriter, notifier cms.Notifier, configError error, upServices ...cluster.Service) {
	r := mux.NewRouter()
	methodNotAllowed := resources.MethodNotAllowed()

	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	r.HandleFunc(httphandlers.GTGPath, resources.GTG(mongo, s3rw, notifier, sched, configError, upServices...)).Methods("GET")
	r.HandleFunc("/__health", resources.Health(mongo, s3rw, notifier, sched, configError, upServices...)).Methods("GET")

	r.HandleFunc("/cycles", resources.GetCycles(sched)).Methods("GET")
	r.HandleFunc("/cycles", resources.CreateCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles", methodNotAllowed).Methods("PUT", "DELETE")

	r.HandleFunc("/cycles/{id}", resources.GetCycleForID(sched)).Methods("GET")
	r.HandleFunc("/cycles/{id}", resources.DeleteCycle(sched)).Methods("DELETE")
	r.HandleFunc("/cycles/{id}", methodNotAllowed).Methods("PUT", "POST")

	r.HandleFunc("/cycles/{id}/resume", resources.ResumeCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/resume", methodNotAllowed).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/cycles/{id}/stop", resources.StopCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/stop", methodNotAllowed).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/cycles/{id}/reset", resources.ResetCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/reset", methodNotAllowed).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/cycles/{id}", resources.DeleteCycle(sched)).Methods("DELETE")
	r.HandleFunc("/cycles/{id}", resources.GetCycleForID(sched)).Methods("GET")

	r.HandleFunc("/scheduler/start", resources.StartScheduler(sched)).Methods("POST")
	r.HandleFunc("/scheduler/start", methodNotAllowed).Methods("GET", "PUT", "DELETE")

	r.HandleFunc("/scheduler/shutdown", resources.ShutdownScheduler(sched)).Methods("POST")
	r.HandleFunc("/scheduler/shutdown", methodNotAllowed).Methods("GET", "PUT", "DELETE")

	box := ui.UI()
	dist := http.FileServer(box.HTTPBox())
	r.PathPrefix("/").Handler(dist)

	http.Handle("/", r)
	log.Info("Publish Carousel Started!")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.WithError(err).Panic("Couldn't set up HTTP listener")
	}
}
