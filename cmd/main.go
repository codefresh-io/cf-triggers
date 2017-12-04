package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/codefresh-io/cf-triggers/pkg/backend"
	"github.com/codefresh-io/cf-triggers/pkg/controller"
	"github.com/codefresh-io/cf-triggers/pkg/version"
)

func main() {
	app := cli.NewApp()
	app.Name = "Codefresh Triggers"
	app.Version = version.HumanVersion
	app.Usage = "Codefresh Trigger configuration and service"
	app.UsageText = fmt.Sprintf(`
Codefresh Triggers allows to configure (through CLI) and serve triggers for Codefresh pipelines.
%s
Codefresh Triggers respects following environment variables:
   - REDIS_HOST         - set the url to the docker serve (default localhost)
   - REDIS_PASSWORD     - set the version of the API to reach
   
Copyright © Codefresh.io`, version.ASCIILogo)
	app.Before = before

	app.Commands = []cli.Command{
		{
			Name: "server",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "codefresh, cf",
					Usage:  "Codefresh API endpoint",
					Value:  "https://g.codefresh.io/api",
					EnvVar: "CFAPI_URL",
				},
				cli.StringFlag{
					Name:   "token, t",
					Usage:  "Codefresh API token",
					EnvVar: "CFAPI_TOKEN",
				},
				cli.IntFlag{
					Name:  "port",
					Usage: "port the trigger manager should serve triggers on",
					Value: 9000,
				},
			},
			Usage:       "start server",
			ArgsUsage:   "configuration file",
			Description: "run trigger manager server",
			Action:      runServer,
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "redis",
			Usage:  "redis host name",
			Value:  "localhost",
			EnvVar: "REDIS_HOST",
		},
		cli.StringFlag{
			Name:   "redis-password",
			Usage:  "redis password",
			EnvVar: "REDIS_PASSWORD",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug mode with verbose logging",
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "do not execute commands, just log",
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "produce log in JSON format: Logstash and Splunk friendly",
		},
	}

	app.Run(os.Args)
}

func before(c *cli.Context) error {
	// set debug log level
	if c.GlobalBool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	// set log formatter to JSON
	if c.GlobalBool("json") {
		log.SetFormatter(&log.JSONFormatter{})
	}

	return nil
}

func runServer(c *cli.Context) {
	fmt.Println(version.ASCIILogo)
	router := gin.Default()

	//triggerController := controller.NewController(backend.NewMemoryStore())
	triggerController := controller.NewController(backend.NewRedisStore(c.GlobalString("redis"), c.GlobalString("redis-password")))

	router.Handle("GET", "/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/triggers")
	})
	router.Handle("GET", "/triggers", triggerController.List)
	router.Handle("GET", "/triggers/:id", triggerController.Get)
	router.Handle("POST", "/triggers", triggerController.Add)
	router.Handle("PUT", "/triggers/:id", triggerController.Update)
	router.Handle("DELETE", "/triggers/:id", triggerController.Delete)

	router.Run()
}
