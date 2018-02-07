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
			Name: "list",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "event",
					Usage: "trigger event filter (cannot be mixed with 'pipeline')",
				},
				cli.StringSliceFlag{
					Name:  "pipeline",
					Usage: "pipeline filter (cannot be mixed with 'event')",
				},
			},
			Usage:       "list defined triggers",
			Description: "List triggers filtered by trigger event(s) or pipeline(s)",
			Action:      listTriggers,
		},
		{
			Name: "run",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "var",
					Usage: "variable pairs (key=val); can pass multiple pairs",
				},
			},
			Usage:       "execute trigger",
			ArgsUsage:   "<event-uri>",
			Description: "Execute trigger for trigger event. Pass multiple variable pairs (key=value), using --var flags.",
			Action:      runTrigger,
		},
	},
}

// get triggers by name(s), filter or ALL
func listTriggers(c *cli.Context) error {
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get events or pipelines
	events := c.StringSlice("event")
	pipelines := c.StringSlice("pipeline")

	// triggers slice
	var err error
	var triggers []model.Trigger

	// Handle 'pipelines'
	if pipelines != nil {
		if events != nil {
			return errors.New("'event' filter cannot be mixed with 'pipeline'")
		}
		triggers, err = triggerReaderWriter.ListTriggersForPipelines(pipelines)
		if err != nil {
			return err
		}
	}
	if events != nil {
		if pipelines != nil {
			return errors.New("'pipeline' filter cannot be mixed with 'event'")
		}
		triggers, err = triggerReaderWriter.ListTriggersForEvents(events)
		if err != nil {
			return err
		}
	}

	if len(triggers) == 0 {
		return errors.New("no triggers defined")
	}
	for _, t := range triggers {
		fmt.Println(t)
	}
	return nil
}

// run all pipelines connected to specified trigger
func runTrigger(c *cli.Context) error {
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("c"), c.GlobalString("t"))
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService, nil)
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
	pipelines, err := triggerReaderWriter.GetPipelinesForTriggers([]string{c.Args().First()})
	if err != nil {
		return err
	}
	// run pipelines
	runs, err := runner.Run(pipelines, vars)
	if err != nil {
		return err
	}

	// print out runs or errors
	for _, r := range runs {
		if r.Error != nil {
			fmt.Println("\terror: ", r.Error)
		} else {
			fmt.Println("\trun: ", r.ID)
		}
	}
	return nil
}
