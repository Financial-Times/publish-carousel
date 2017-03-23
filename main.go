package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ui "github.com/Financial-Times/publish-carousel-ui"
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
			Name:  "cycles",
			Value: "./cycles.yml",
			Usage: "Path to the YML cycle configuration file.",
		},
		cli.StringFlag{
			Name:   "mongo-db",
			Value:  "localhost:27017",
			EnvVar: "MONGO_DB_URL",
			Usage:  "The Mongo DB connection url string (comma delimited).",
		},
		cli.StringFlag{
			Name:   "cms-notifier-url",
			Value:  "http://localhost:8080/__cms-notifier/notify",
			EnvVar: "CMS_NOTIFIER_URL",
			Usage:  "The CMS Notifier instance to POST publishes to.",
		},
		cli.StringFlag{
			Name:   "cms-notifier-gtg",
			Value:  "http://localhost:8080/__cms-notifier/__gtg",
			EnvVar: "CMS_NOTIFIER_GTG",
			Usage:  "The CMS Notifier GTG url.",
		},
		cli.StringFlag{
			Name:   "aws-region",
			Value:  "eu-west-1",
			EnvVar: "AWS_REGION",
			Usage:  "The AWS Region for this cluster.",
		},
		cli.StringFlag{
			Name:   "s3-bucket",
			Value:  "com.ft.universalpublishing.publish-carousel.dynpub-uk",
			EnvVar: "S3_BUCKET",
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
	}

	app.Action = func(ctx *cli.Context) {
		log.Info("Starting the Publish Carousel.")

		s3rw := s3.NewReadWriter(ctx.String("aws-region"), ctx.String("s3-bucket"))
		stateRw := scheduler.NewS3MetadataReadWriter(s3rw)

		mongo := native.NewMongoDatabase(ctx.String("mongo-db"), ctx.Int("mongo-timeout"))

		reader := native.NewMongoNativeReader(mongo)
		notifier := cms.NewNotifier(ctx.String("cms-notifier-url"), ctx.String("cms-notifier-gtg"), &http.Client{Timeout: time.Second * 30})

		task := tasks.NewNativeContentPublishTask(reader, notifier)

		etcdWatcher, err := etcd.NewEtcdWatcher(ctx.StringSlice("etcd-peers"))

		if err != nil {
			panic(err)
		}

		sched, configError := scheduler.LoadSchedulerFromFile(ctx.String("cycles"), mongo, task, stateRw)
		if err != nil {
			log.WithError(configError).Error("Failed to load cycles configuration file")
		}

		toggle, err := etcdWatcher.Read(ctx.String("toggle-etcd-key"))
		if err != nil {
			panic(err)
		}

		sched.ToggleHandler(toggle)

		go etcdWatcher.Watch(ctx.String("toggle-etcd-key"), sched.ToggleHandler)

		sched.RestorePreviousState()
		sched.Start()

		shutdown(sched)
		serve(mongo, sched, s3rw, notifier, configError)
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

func serve(mongo native.DB, sched scheduler.Scheduler, s3rw s3.ReadWriter, notifier cms.Notifier, configError error) {
	r := mux.NewRouter()
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	r.HandleFunc(httphandlers.GTGPath, resources.GTG(mongo, s3rw, notifier, sched, configError)).Methods("GET")
	r.HandleFunc("/__health", resources.Health(mongo, s3rw, notifier, sched, configError)).Methods("GET")

	r.HandleFunc("/cycles", resources.GetCycles(sched)).Methods("GET")
	r.HandleFunc("/cycles", resources.CreateCycle(sched)).Methods("POST")

	r.HandleFunc("/cycles/{id}", resources.GetCycleForID(sched)).Methods("GET")
	r.HandleFunc("/cycles/{id}", resources.DeleteCycle(sched)).Methods("DELETE")

	r.HandleFunc("/cycles/{id}/resume", resources.ResumeCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/stop", resources.StopCycle(sched)).Methods("POST")
	r.HandleFunc("/cycles/{id}/reset", resources.ResetCycle(sched)).Methods("POST")

	r.HandleFunc("/scheduler/start", resources.StartScheduler(sched)).Methods("POST")
	r.HandleFunc("/scheduler/shutdown", resources.ShutdownScheduler(sched)).Methods("POST")

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
