package main

import (
	"net/http"
	"os"
	"time"

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
	app.Action = func() {
		serve()
	}

	app.Run(os.Args)
}

func serve() {
	r := mux.NewRouter()
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	http.Handle("/", r)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.WithError(err).Panic("Couldn't set up HTTP listener")
	}
}
