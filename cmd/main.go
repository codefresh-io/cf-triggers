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
		{
			Name:  "trigger",
			Usage: "manage triggers",
			Subcommands: []cli.Command{
				{
					Name:        "get",
					Usage:       "get triggers",
					ArgsUsage:   "trigger id or empty (ALL)",
					Description: "get specific trigger or all trigger",
					Action:      getTriggers,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "redis",
			Usage:  "redis host name",
			Value:  "localhost",
			EnvVar: "REDIS_HOST",
		},
		cli.IntFlag{
			Name:   "redis-port",
			Usage:  "redis host port",
			Value:  6379,
			EnvVar: "REDIS_PORT",
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

func runServer(c *cli.Context) error {
	fmt.Println(version.ASCIILogo)
	router := gin.Default()

	//triggerController := controller.NewController(backend.NewMemoryStore())
	triggerController := controller.NewController(backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password")))

	router.Handle("GET", "/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/triggers")
	})
	router.Handle("GET", "/triggers", triggerController.List)
	router.Handle("GET", "/triggers/:id", triggerController.Get)
	router.Handle("POST", "/triggers", triggerController.Add)
	router.Handle("PUT", "/triggers/:id", triggerController.Update)
	router.Handle("DELETE", "/triggers/:id", triggerController.Delete)

	return router.Run()
}

func getTriggers(c *cli.Context) error {
	// triggerService := backend.NewMemoryStore()
	triggerService := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"))
	if len(c.Args()) == 0 {
		triggers, err := triggerService.List()
		if err != nil {
			log.Error(err)
			return err
		}
		if len(triggers) == 0 {
			fmt.Println("No triggers defined!")
		}
		for _, t := range triggers {
			fmt.Printf("%+v\n", t)
		}
	} else {
		for _, id := range c.Args() {
			trigger, err := triggerService.Get(id)
			if err != nil {
				log.Error(err)
				return err
			}
			if trigger.IsEmpty() {
				fmt.Printf("Trigger '%s' not found!\n", id)
			} else {
				fmt.Printf("%+v\n", trigger)
			}
		}
	}

	return nil
}
