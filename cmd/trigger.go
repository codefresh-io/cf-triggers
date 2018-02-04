package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/urfave/cli"
)

var triggerCommand = cli.Command{
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
				cli.StringFlag{
					Name:  "pipeline, p",
					Usage: "additional filter by pipeline UID (ignored when using '--filter')",
				},
				cli.BoolFlag{
					Name:  "quiet, q",
					Usage: "only display event URIs",
				},
			},
			Usage:       "get trigger(s)",
			ArgsUsage:   "[name, filter or empty (ALL)]",
			Description: "Get trigger by name or filter, or get all triggers, if no filter specified",
			Action:      getTriggers,
		},
		{
			Name: "add",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "secret, s",
					Usage: "trigger secret (auto-generated if skipped)",
					Value: model.GenerateKeyword,
				},
			},
			Usage:       "add trigger",
			ArgsUsage:   "<event URI> <pipeline UID>",
			Description: "Add a new trigger connected to specified pipeline",
			Action:      addTrigger,
		},
		{
			Name:        "delete",
			Usage:       "delete trigger",
			ArgsUsage:   "<event URI>",
			Description: "Delete trigger by event URI",
			Action:      deleteTrigger,
		},
		{
			Name: "test",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "var",
					Usage: "variable pairs (key=val); can pass multiple pairs",
				},
			},
			Usage:       "trigger pipeline execution with variables",
			ArgsUsage:   "[name]",
			Description: "Invoke trigger, specified by trigger name. Can pass multiple variable pairs (key=value), using --var flags.",
			Action:      testTrigger,
		},
	},
}

// get triggers by name(s), filter or ALL
func getTriggers(c *cli.Context) error {
	quiet := c.Bool("quiet")
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	filter := c.String("filter")
	pipelineUID := c.String("pipeline")
	// Handle 'pipeline'
	if pipelineUID != "" {
		if filter != "" {
			return fmt.Errorf("pipeline cannot be used with filter")
		}
		triggers, err := triggerReaderWriter.ListByPipeline(pipelineUID)
		if err != nil {
			return err
		}
		if len(triggers) == 0 {
			return fmt.Errorf("no triggers defined")
		}
		for _, t := range triggers {
			if quiet {
				fmt.Println(t.Event)
			} else {
				fmt.Println(t)
			}
		}
		return nil
	}
	// Handle 'filter'
	if len(c.Args()) == 0 {
		triggers, err := triggerReaderWriter.List(filter)
		if err != nil {
			return err
		}
		if len(triggers) == 0 {
			return fmt.Errorf("no triggers defined")
		}
		for _, t := range triggers {
			if quiet {
				fmt.Println(t.Event)
			} else {
				fmt.Println(t)
			}
		}
	} else {
		for _, id := range c.Args() {
			trigger, err := triggerReaderWriter.Get(id)
			if err != nil {
				return err
			}
			if quiet {
				fmt.Println(trigger.Event)
			} else {
				fmt.Println(trigger)
			}
		}
	}

	return nil
}

// add new trigger
func addTrigger(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 2 {
		return errors.New("wrong number of arguments")
	}
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("c"), c.GlobalString("t"))
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService)
	// create trigger model
	trigger := model.Trigger{}
	trigger.Event = args.First()
	trigger.Secret = c.String("secret")
	trigger.Pipelines = make([]string, 1)
	trigger.Pipelines[0] = args.Get(1)
	return triggerReaderWriter.Add(trigger)
}

// add new trigger
func deleteTrigger(c *cli.Context) error {
	// get trigger name
	args := c.Args()
	if len(args) != 1 {
		return errors.New("wrong argument, expected trigger event URI")
	}
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	return triggerReaderWriter.Delete(args.First())
}

// run all pipelines connected to specified trigger
func testTrigger(c *cli.Context) error {
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("c"), c.GlobalString("t"))
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService)
	// get pipeline runner
	runner := backend.NewRunner(codefreshService)
	// convert command line 'var' variables (key=value) to map
	vars := make(map[string]string)
	for _, v := range c.StringSlice("var") {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			return fmt.Errorf("Invalid 'var' value: %s ; should be 'key=value' form", v)
		}
		vars[kv[0]] = kv[1]
	}

	// get trigger pipelines
	pipelines, err := triggerReaderWriter.GetPipelines([]string{c.Args().First()})
	if err != nil {
		return err
	}
	// run pipelines
	runs, err := runner.Run(pipelines, vars)
	if err != nil {
		return err
	}

	// print out runs
	for _, r := range runs {
		if r.Error != nil {
			fmt.Println("\terror: ", r.Error)
		} else {
			fmt.Println("\trun: ", r.ID)
		}
	}
	return nil
}
