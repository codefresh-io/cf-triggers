package main

import (
	"errors"
	"fmt"

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
			Name:        "link",
			Usage:       "connect trigger event to the specified pipeline(s)",
			ArgsUsage:   "<event-uri> <pipeline> [pipeline...]",
			Description: "Create a new trigger, linking a trigger event to the specified pipeline(s)",
			Action:      linkEvent,
		},
		{
			Name:        "unlink",
			Usage:       "disconnect trigger event from the specified pipeline(s)",
			ArgsUsage:   "<event-uri> <pipeline> [pipeline...]",
			Description: "Delete trigger, by removing link between the trigger event and the specified pipeline(s)",
			Action:      unlinkEvent,
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

	// at least one option must be defined
	if len(events) == 0 && len(pipelines) == 0 {
		return errors.New("at least one option 'event' or 'pipeline' must be specified")
	}
	// Handle 'pipelines'
	if len(pipelines) > 0 {
		if len(events) > 0 {
			return errors.New("'event' filter cannot be mixed with 'pipeline'")
		}
		triggers, err = triggerReaderWriter.ListTriggersForPipelines(pipelines)
		if err != nil {
			return err
		}
	}
	if len(events) > 0 {
		if len(pipelines) > 0 {
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

func linkEvent(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) < 2 {
		return errors.New("wrong number of arguments")
	}
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("c"), c.GlobalString("t"))
	// get trigger service
	eventReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService, nil)
	// create triggers for event linking it to passed pipeline(s)
	return eventReaderWriter.CreateTriggersForEvent(c.Args().First(), c.Args().Tail())
}

func unlinkEvent(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 2 {
		return errors.New("wrong number of arguments")
	}
	// get trigger service
	eventReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// delete pipelines
	return eventReaderWriter.DeleteTriggersForEvent(args.First(), args.Tail())
}
