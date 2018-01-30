package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ui "github.com/Financial-Times/publish-carousel-ui"
	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/cluster"
	cluster_etcd "github.com/Financial-Times/publish-carousel/cluster/etcd"
	cluster_file "github.com/Financial-Times/publish-carousel/cluster/file"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/etcd"
	"github.com/Financial-Times/publish-carousel/file"
	"github.com/Financial-Times/publish-carousel/image"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/resources"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/Financial-Times/publish-carousel/tasks"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/husobee/vestigo"
	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

const (
	appSystemCode = "publish-carousel"
	appName       = "UPP Publish Carousel"
	description   = "A microservice that continuously republishes content and annotations available in the native store."
)

func init() {
	f := &log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}

	log.SetLevel(log.InfoLevel)
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
		cli.IntFlag{
			Name:   "mongo-node-count",
			Value:  1,
			EnvVar: "MONGO_NODE_COUNT",
			Usage:  "The number of Mongo DB instances.",
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
			Usage:  "The URL of the queue lagcheck service from the publishing cluster to verify the health of the cluster.",
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
		cli.StringFlag{
			Name:   "api-yml",
			EnvVar: "API_YML",
			Value:  "./api.yml",
			Usage:  "The swagger API yaml file for the service.",
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
			Usage:  "The etcd key that enables or disables the carousel",
		},
		cli.StringFlag{
			Name:   "read-monitoring-etcd-key",
			Value:  "/ft/config/monitoring/read-urls",
			EnvVar: "READ_URLS_ETCD_KEY",
			Usage:  "The etcd key which contains all the read environments",
		},
		cli.StringFlag{
			Name:   "active-cluster-etcd-key",
			Value:  "/ft/healthcheck-categories/publish/enabled",
			EnvVar: "ACTIVE_CLUSTER_ETCD_KEY",
			Usage:  "The ETCD key that specifies if the cluster is active",
		},
		cli.StringFlag{
			Name:   "default-throttle",
			Value:  "1m",
			EnvVar: "DEFAULT_THROTTLE",
			Usage:  "Default throttle for whole collection cycles, if it is not specified in configuration file",
		},
		cli.StringFlag{
			Name:   "checkpoint-interval",
			Value:  "1h",
			EnvVar: "CHECKPOINT_INTERVAL",
			Usage:  "Interval for saving metadata checkpoints",
		},
		cli.StringFlag{
			Name:   "configs-dir",
			Value:  "/configs",
			EnvVar: "CONFIGS_DIR",
			Usage:  "Directory containing the files with read environment, toggle and active-cluster values",
		},
		cli.StringFlag{
			Name:   "credentials-dir",
			Value:  "/configs/credentials",
			EnvVar: "CREDENTIALS_DIR",
			Usage:  "Directory containing the file with read environment credentials",
		},
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	app.Action = func(ctx *cli.Context) {
		log.Info("Starting the Publish Carousel.")

		if err := native.CheckMongoURLs(ctx.String("mongo-db"), ctx.Int("mongo-node-count")); err != nil {
			panic(fmt.Sprintf("Provided MongoDB URLs are invalid: %s", err))
		}

		s3rw := s3.NewReadWriter(ctx.String("aws-region"), ctx.String("s3-bucket"))
		stateRw := scheduler.NewS3MetadataReadWriter(s3rw)

		isImage := image.NewFilter()
		blacklist, err := blacklist.NewFileBasedBlacklist(ctx.String("blacklist"))
		if err != nil {
			panic(err)
		}

		client := &http.Client{Timeout: time.Second * 30}

		mongo := native.NewMongoDatabase(ctx.String("mongo-db"), ctx.Int("mongo-timeout"))

		reader := native.NewMongoNativeReader(mongo)
		notifier, err := cms.NewNotifier(ctx.String("cms-notifier-url"), client)
		if err != nil {
			log.WithError(err).Error("Error in CMS Notifier configuration")
		}

		pam, err := cluster.NewService("publish-availability-monitor", ctx.String("pam-url"), true) // true so that we check /__health
		if err != nil {
			log.WithError(err).Error("Error in Publish Availability Monitor configuration")
		}

		publishingLagcheck, err := cluster.NewService("kafka-lagcheck", ctx.String("lagcheck-url"), false)
		if err != nil {
			panic(err)
		}

		task := tasks.NewNativeContentPublishTask(reader, notifier, isImage)

		defaultThrottle, err := time.ParseDuration(ctx.String("default-throttle"))
		if err != nil {
			log.WithError(err).Error("Invalid value for default throttle")
		}

		checkpointInterval, err := time.ParseDuration(ctx.String("checkpoint-interval"))
		if err != nil {
			log.WithError(err).Error("Invalid checkpoint interval, defaulting to hourly.")
			checkpointInterval = time.Hour
		}

		uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(mongo, s3rw, blacklist)

		sched, configError := scheduler.LoadSchedulerFromFile(ctx.String("cycles"), uuidCollectionBuilder, task, stateRw, defaultThrottle, checkpointInterval)
		if configError != nil {
			log.WithError(configError).Error("Failed to load cycles configuration file")
		}

		var deliveryLagcheck cluster.Service
		var manualToggle, autoToggle string

		if ctx.StringSlice("etcd-peers")[0] == "NOT_AVAILABLE" {
			log.Info("Sourcing configs from file.")
			fileWatcher, err := file.NewFileWatcher([]string{ctx.String("configs-dir"), ctx.String("credentials-dir")}, time.Second*30)
			if err != nil {
				panic(err)
			}
			deliveryLagcheck, err = cluster_file.NewExternalService("kafka-lagcheck-delivery", client, "kafka-lagcheck", fileWatcher, "read.environments", "read.credentials")
			if err != nil {
				panic(err)
			}
			manualToggle, _ = fileWatcher.Read("toggle")
			autoToggle, _ = fileWatcher.Read("active-cluster")

			log.WithField("manualToggle", manualToggle).WithField("autoToggle", autoToggle).Info("Read configs!")

			go fileWatcher.Watch(context.Background(), "toggle", sched.ManualToggleHandler)
			go fileWatcher.Watch(context.Background(), "active-cluster", sched.AutomaticToggleHandler)
		} else {
			log.Info("Sourcing configs from etcd.")
			etcdWatcher, err := etcd.NewEtcdWatcher(ctx.StringSlice("etcd-peers"))
			if err != nil {
				panic(err)
			}

			deliveryLagcheck, err = cluster_etcd.NewExternalService("kafka-lagcheck-delivery", client, "kafka-lagcheck", etcdWatcher, ctx.String("read-monitoring-etcd-key"))
			if err != nil {
				panic(err)
			}

			manualToggle, err = etcdWatcher.Read(ctx.String("toggle-etcd-key"))
			if err != nil {
				panic(err)
			}
			autoToggle, err = etcdWatcher.Read(ctx.String("active-cluster-etcd-key"))
			if err != nil {
				panic(err)
			}
			go etcdWatcher.Watch(context.Background(), ctx.String("toggle-etcd-key"), sched.ManualToggleHandler)
			go etcdWatcher.Watch(context.Background(), ctx.String("active-cluster-etcd-key"), sched.AutomaticToggleHandler)

		}

		sched.ManualToggleHandler(manualToggle)
		sched.AutomaticToggleHandler(autoToggle)
		sched.RestorePreviousState()
		sched.Start()

		api, _ := ioutil.ReadFile(ctx.String("api-yml"))

		shutdown(sched)
		serve(mongo, sched, s3rw, notifier, api, configError, pam, publishingLagcheck, deliveryLagcheck)
	}

	app.Run(os.Args)
}

func shutdown(sched scheduler.Scheduler) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		signal := <-signals
		log.WithField("signal", signal).Info("Stopping scheduler after receiving OS signal")
		err := sched.Shutdown()
		if err != nil {
			log.WithError(err).Error("Error in stopping scheduler")
		}
		os.Exit(0)
	}()
}

func serve(mongo native.DB, sched scheduler.Scheduler, s3rw s3.ReadWriter, notifier cms.Notifier, api []byte, configError error, upServices ...cluster.Service) {
	r := vestigo.NewRouter()

	healthService := resources.NewHealthService(appSystemCode, appName, description, mongo, s3rw, notifier, sched, configError, upServices...)

	r.Get("/__api", resources.API(api))
	r.Post("/__log", resources.LogLevel)

	r.Get(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler)
	r.Get(httphandlers.PingPath, httphandlers.PingHandler)

	r.Get(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(healthService.GTG))
	r.Get("/__health", healthService.Health())

	r.Get("/cycles", resources.GetCycles(sched))
	r.Post("/cycles", resources.CreateCycle(sched))

	r.Get("/cycles/:id", resources.GetCycleForID(sched))
	r.Delete("/cycles/:id", resources.DeleteCycle(sched))

	r.Get("/cycles/:id/throttle", resources.GetCycleThrottle(sched))
	r.Put("/cycles/:id/throttle", resources.SetCycleThrottle(sched))

	r.Post("/cycles/:id/resume", resources.ResumeCycle(sched))

	r.Post("/cycles/:id/stop", resources.StopCycle(sched))

	r.Post("/cycles/:id/reset", resources.ResetCycle(sched))

	r.Post("/scheduler/start", resources.StartScheduler(sched))

	r.Post("/scheduler/shutdown", resources.ShutdownScheduler(sched))

	box := ui.UI()
	dist := http.FileServer(box.HTTPBox())
	r.Get("/*", dist.ServeHTTP)

	http.Handle("/", r)
	log.Info("Publish Carousel Started!")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.WithError(err).Panic("Couldn't set up HTTP listener")
	}
}
