package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/controller"
	"github.com/codefresh-io/hermes/pkg/version"
)

func main() {
	app := cli.NewApp()
	app.Name = "hermes"
	app.Authors = []cli.Author{{Name: "Alexei Ledenev", Email: "alexei@codefresh.io"}}
	app.Version = version.HumanVersion
	app.EnableBashCompletion = true
	app.Usage = "configure triggers and run trigger manager server"
	app.UsageText = fmt.Sprintf(`Configure triggers for Codefresh pipeline execution or start trigger manager server. Process "normalized" events and run Codefresh pipelines with variables extracted from events payload.
%s
hermes respects following environment variables:
   - REDIS_HOST         - set the url to the Redis server (default localhost)
   - REDIS_PORT         - set Redis port (default to 6379)
   - REDIS_PASSWORD     - set Redis password
   
Copyright © Codefresh.io`, version.ASCIILogo)
	app.Before = before

	app.Commands = []cli.Command{
		{
			Name: "server",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "codefresh, cf",
					Usage:  "Codefresh API endpoint",
					Value:  "https://g.codefresh.io/",
					EnvVar: "CFAPI_URL",
				},
				cli.StringFlag{
					Name:   "token, t",
					Usage:  "Codefresh API token",
					EnvVar: "CFAPI_TOKEN",
				},
				cli.IntFlag{
					Name:  "port",
					Usage: "TCP port for the trigger manager server",
					Value: 9000,
				},
			},
			Usage:       "start trigger manager server",
			Description: "Run Codefresh trigger manager server. Use REST API to manage triggers. Send normalized event payload to trigger endpoint to invoke associated Codefresh pipelines.",
			Action:      runServer,
		},
		{
			Name:  "trigger",
			Usage: "configure Codefresh triggers",
			Subcommands: []cli.Command{
				{
					Name: "get",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "filter, f",
							Usage: "trigger filter",
						},
					},
					Usage:       "get defined trigger(s)",
					ArgsUsage:   "[name, filter or empty (ALL)]",
					Description: "Get trigger by name or filter, or get all triggers, if no filter specified",
					Action:      getTriggers,
				},
				{
					Name: "test",
					Flags: []cli.Flag{
						cli.StringSliceFlag{
							Name:  "var",
							Usage: "variable pairs (key=val); can pass multiple pairs",
						},
						cli.StringFlag{
							Name:   "codefresh, cf",
							Usage:  "Codefresh API endpoint",
							Value:  "https://g.codefresh.io/",
							EnvVar: "CFAPI_URL",
						},
						cli.StringFlag{
							Name:   "token, t",
							Usage:  "Codefresh API token",
							EnvVar: "CFAPI_TOKEN",
						},
					},
					Usage:       "trigger pipeline execution with variables",
					ArgsUsage:   "[name]",
					Description: "Invoke trigger, specified by trigger name. Can pass multiple variable pairs (key=value), using --var flags.",
					Action:      testTrigger,
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

// start trigger manager server
func runServer(c *cli.Context) error {
	fmt.Println(version.ASCIILogo)
	router := gin.Default()

	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.String("cf"), c.String("t"))

	//triggerController := controller.NewController(backend.NewMemoryStore(codefresh.PipelineService))
	triggerController := controller.NewController(backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService))

	// trigger management API
	router.Handle("GET", "/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/triggers")
	})
	router.Handle("GET", "/triggers/", triggerController.List) // pass filter as query parameter
	router.Handle("GET", "/triggers/:id", triggerController.Get)
	router.Handle("POST", "/triggers", triggerController.Add)
	router.Handle("PUT", "/triggers/:id", triggerController.Update)
	router.Handle("DELETE", "/triggers/:id", triggerController.Delete)
	// invoke trigger with event payload
	router.Handle("POST", "/trigger/:id", triggerController.Run)

	return router.Run()
}

// get triggers by name(s), filter or ALL
func getTriggers(c *cli.Context) error {
	// triggerService := backend.NewMemoryStore(codefresh.PipelineService)
	triggerService := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	if len(c.Args()) == 0 {
		triggers, err := triggerService.List(c.String("filter"))
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

// run all pipelines connected to specified trigger
func testTrigger(c *cli.Context) error {
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.String("cf"), c.String("t"))
	// get trigger service
	triggerService := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService)
	// convert command line 'var' variables (key=value) to map
	vars := make(map[string]string)
	for _, v := range c.StringSlice("var") {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			return fmt.Errorf("Invalid 'var' value: %s ; should be 'key=value' form", v)
		}
		vars[kv[0]] = kv[1]
	}

	// get trigger from argument
	return triggerService.Run(c.Args().First(), vars)
}
