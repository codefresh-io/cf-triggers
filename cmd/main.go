package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/codefresh-io/hermes/pkg/logger"
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
   - STORE_HOST         - set the url to the Redis store server (default localhost)
   - STORE_PORT         - set Redis store port (default to 6379)
   - STORE_PASSWORD     - set Redis store password
   
Copyright © Codefresh.io`, version.ASCIILogo)
	app.Before = before

	app.Commands = []cli.Command{
		serverCommand,
		triggerCommand,
		runnerCommand,
		triggerEventCommand,
		triggerTypeCommand,
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "codefresh, c",
			Usage:  "Codefresh API endpoint",
			Value:  "http://local.codefresh.io:9007",
			EnvVar: "CFAPI_URL",
		},
		cli.StringFlag{
			Name:   "token, t",
			Usage:  "Codefresh API token",
			EnvVar: "CFAPI_TOKEN",
		},
		cli.StringFlag{
			Name:   "redis, r",
			Usage:  "redis store host name",
			Value:  "local.codefresh.io",
			EnvVar: "STORE_HOST",
		},
		cli.IntFlag{
			Name:   "redis-port, p",
			Usage:  "redis store port",
			Value:  6379,
			EnvVar: "STORE_PORT",
		},
		cli.StringFlag{
			Name:   "redis-password, s",
			Usage:  "redis store password",
			Value:  "redisPassword",
			EnvVar: "STORE_PASSWORD",
		},
		cli.StringFlag{
			Name:   "config",
			Usage:  "type config file",
			Value:  "./pkg/backend/dev_external_types.json",
			EnvVar: "TYPES_CONFIG",
		},
		cli.BoolFlag{
			Name:   "skip-monitor, m",
			Usage:  "skip monitoring config file for changes",
			EnvVar: "SKIP_MONITOR",
		},
		cli.StringFlag{
			Name:   "log-level, l",
			Usage:  "set log level (debug, info, warning(*), error, fatal, panic)",
			Value:  "warning",
			EnvVar: "LOG_LEVEL",
		},
		cli.BoolFlag{
			Name:  "dry-run, x",
			Usage: "do not execute commands, just log",
		},
		cli.BoolFlag{
			Name:   "json, j",
			Usage:  "produce log in JSON format: Codefresh friendly",
			EnvVar: "LOG_JSON",
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func before(c *cli.Context) error {
	// set debug log level
	switch level := c.GlobalString("log-level"); level {
	case "debug", "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "info", "INFO":
		log.SetLevel(log.InfoLevel)
	case "warning", "WARNING":
		log.SetLevel(log.WarnLevel)
	case "error", "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "fatal", "FATAL":
		log.SetLevel(log.FatalLevel)
	case "panic", "PANIC":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}
	// set log formatter to Codefresh JSON
	if c.GlobalBool("json") {
		log.SetFormatter(&logger.CFFormatter{})
	}
	// trace function calls
	traceHook := logger.NewHook()
	traceHook.Prefix = "codefresh:hermes:"
	traceHook.AppName = "hermes"
	traceHook.FunctionField = logger.FieldNamespace
	traceHook.AppField = logger.FieldService
	log.AddHook(traceHook)

	return nil
}
