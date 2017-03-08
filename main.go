package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Financial-Times/publish-carousel/cms"
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
			Name:   "aws-region",
			Value:  "eu-west-1",
			EnvVar: "AWS_REGION",
			Usage:  "The AWS Region for this cluster.",
		},
		cli.StringFlag{
			Name:   "s3-bucket",
			Value:  "/publish/carousel",
			EnvVar: "S3_BUCKET",
			Usage:  "The S3 Bucket to save carousel states.",
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

		s3api, err := s3.NewS3ReadWrite(ctx.String("aws-region"), ctx.String("s3-bucket"))

		state := restorePreviousState(s3api)

		mongo := native.NewMongoDatabase(ctx.String("mongo-db"), ctx.Int("mongo-timeout"))

		reader := native.NewMongoNativeReader(mongo)
		notifier := cms.NewNotifier()

		task := tasks.NewNativeContentPublishTask(reader, notifier)

		sched, _ := scheduler.LoadSchedulerFromFile(ctx.String("cycles"), mongo, task) //TODO: do something with this error
		serve(mongo, sched)
	}

	app.Run(os.Args)
}

func restorePreviousState(s3api s3.S3ReadWrite) *scheduler.CycleState {
	id, err := s3api.GetLatestID()
	if err != nil {
		log.WithError(err).Warn("Failed to retrieve carousel state from S3 - starting from initial state.")
	}

	found, state, contentType, err := s3api.Read(id)
	if err != nil || !found {
		log.WithField("id", id).WithError(err).Warn("Failed to read carousel state from S3. Error occurred while reading from ID.")
		return nil
	}

	if contentType != nil && *contentType != "application/json" {
		log.WithField("content-type", contentType).Warn("Failed to read carousel state from S3 - unexpected content type.")
		return nil
	}

	result := &scheduler.CycleState{}
	dec := json.NewDecoder(state)
	dec.Decode(result)
	return result
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
