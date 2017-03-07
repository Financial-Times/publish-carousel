package main

import (
	"net/http"
	"os"
	"time"

	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/resources"
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
		cli.IntFlag{
			Name:   "mongo-timeout",
			Value:  30000,
			EnvVar: "MONGO_DB_TIMEOUT",
			Usage:  "The timeout (in milliseconds) for Mongo DB connections.",
		},
	}

	app.Action = func(ctx *cli.Context) {
		log.Info("Starting the Publish Carousel.")
		mongo := native.NewMongoDatabase(ctx.String("mongo-db"), ctx.Int("mongo-timeout"))

		reader := native.NewMongoNativeReader(mongo)
		notifier := cms.NewNotifier()

		task := tasks.NewNativeContentPublishTask(reader, notifier)

		sched, _ := scheduler.LoadSchedulerFromFile(ctx.String("cycles"), mongo, task) //TODO: do something with this error
		serve(mongo, sched)
	}

	app.Run(os.Args)
}

func serve(mongo native.DB, sched scheduler.Scheduler) {
	r := mux.NewRouter()
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	r.HandleFunc(httphandlers.GTGPath, resources.GTG(mongo)).Methods("GET")
	r.HandleFunc("/__health", resources.Health(mongo)).Methods("GET")

	r.HandleFunc("/cycles", resources.GetCycles(sched)).Methods("GET")

	http.Handle("/", r)
	log.Info("Publish Carousel Started!")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.WithError(err).Panic("Couldn't set up HTTP listener")
	}
}
