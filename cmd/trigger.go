package main

import (
	"errors"
	"fmt"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
	"github.com/urfave/cli"
)

var triggerCommand = cli.Command{
	Name:  "trigger",
	Usage: "configure Codefresh triggers",
	Subcommands: []cli.Command{
		{
			Name: "list",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "event",
					Usage: "trigger event filter (cannot be mixed with 'pipeline')",
				},
				cli.StringFlag{
					Name:  "pipeline",
					Usage: "pipeline filter (cannot be mixed with 'event')",
				},
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
					Value: model.PublicAccount,
				},
			},
			Usage:       "list defined triggers",
			Description: "List triggers filtered by trigger event or pipeline",
			Action:      listTriggers,
		},
		{
			Name: "create",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
					Value: model.PublicAccount,
				},
				cli.StringSliceFlag{
					Name:  "filter",
					Usage: "filter pairs (name=condition); can pass multiple pairs",
				},
			},
			Usage:       "create trigger",
			ArgsUsage:   "<event-uri> <pipeline>",
			Description: "Create a new trigger, linking the trigger event to the specified pipeline",
			Action:      createTrigger,
		},
		{
			Name: "delete",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
					Value: model.PublicAccount,
				},
			},
			Usage:       "delete trigger",
			ArgsUsage:   "<event-uri> <pipeline>",
			Description: "Delete trigger, by removing link between the trigger event and the specified pipeline",
			Action:      deleteTrigger,
		},
	},
}

// get triggers by name(s), filter or ALL
func listTriggers(c *cli.Context) error {
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get event or pipeline
	event := c.String("event")
	pipeline := c.String("pipeline")

	// triggers slice
	var err error
	var triggers []model.Trigger

	// list by event
	if event != "" {
		triggers, err = triggerReaderWriter.GetEventTriggers(getContext(c), event)
		if err != nil {
			return err
		}
	}

	// list by pipeline
	if pipeline != "" {
		triggers, err = triggerReaderWriter.GetPipelineTriggers(getContext(c), pipeline)
		if err != nil {
			return err
		}
	}

	// get all triggers for all events (private and public)
	if event == "" && pipeline == "" {
		triggers, err = triggerReaderWriter.GetEventTriggers(getContext(c), "*")
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

func createTrigger(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 2 {
		return errors.New("wrong number of arguments")
	}
	// convert command line 'filter' variables (key=value) to map
	filters, err := util.StringSliceToMap(c.StringSlice("filter"))
	if err != nil {
		return err
	}
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("c"), c.GlobalString("t"))
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService, nil)
	// create triggers for event linking it to passed pipeline(s)
	return triggerReaderWriter.CreateTrigger(getContext(c), args.First(), args.Get(1), filters)
}

func deleteTrigger(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 2 {
		return errors.New("wrong number of arguments")
	}
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// delete pipelines
	return triggerReaderWriter.DeleteTrigger(getContext(c), args.First(), args.Get(1))
}
