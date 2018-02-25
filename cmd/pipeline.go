package main

import (
	"errors"
	"fmt"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/urfave/cli"
)

var pipelineCommand = cli.Command{
	Name:  "pipeline",
	Usage: "setup Codefresh triggers, linking trigger events and pipelines",
	Subcommands: []cli.Command{
		{
			Name: "list",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "event",
					Usage: "trigger event filter",
				},
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
				},
			},
			Usage:       "list pipelines with triggers",
			Description: "List Codefresh pipelines that have triggers defined",
			Action:      listPipelines,
		},
		{
			Name:        "link",
			Usage:       "connect pipeline to the specified trigger event(s)",
			ArgsUsage:   "<pipeline> <event-uri> [event-uri...]",
			Description: "Create a new trigger, linking a pipeline to the specified trigger event(s)",
			Action:      linkPipeline,
		},
		{
			Name:        "unlink",
			Usage:       "disconnect pipeline from the specified trigger event(s)",
			ArgsUsage:   "<pipeline> <event-uri> [event-uri...]",
			Description: "Delete pipeline trigger, by removing link between the pipeline and the specified trigger event(s)",
			Action:      unlinkPipeline,
		},
	},
}

func listPipelines(c *cli.Context) error {
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)

	events := c.StringSlice("event")
	var pipelines []string
	var err error

	if len(events) == 0 {
		pipelines, err = triggerReaderWriter.GetAllPipelines()
	} else {
		pipelines, err = triggerReaderWriter.GetPipelinesForTriggers(events, c.String("account"))
	}
	if err != nil {
		return err
	}
	for _, p := range pipelines {
		fmt.Println(p)
	}

	return nil
}

func linkPipeline(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) < 2 {
		return errors.New("wrong number of arguments")
	}
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("c"), c.GlobalString("t"))
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService, nil)
	// create triggers for pipeline linking it to passed event(s)
	return triggerReaderWriter.CreateTriggersForPipeline(args.First(), args.Tail())
}

func unlinkPipeline(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) < 2 {
		return errors.New("wrong number of arguments")
	}
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// delete pipelines
	return triggerReaderWriter.DeleteTriggersForPipeline(args.First(), args.Tail())
}
