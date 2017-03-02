package main

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
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
		log.Info("Hello World!")
	}

	app.Run(os.Args)
}
