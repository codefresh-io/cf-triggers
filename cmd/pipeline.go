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
	Usage: "configure Codefresh trigger pipelines",
	Subcommands: []cli.Command{
		{
			Name:        "get",
			Usage:       "get pipelines connected to trigger",
			ArgsUsage:   "<event URI>",
			Description: "Get all pipelines connected to trigger with provided event URI",
			Action:      getTriggerPipelines,
		},
		{
			Name:        "add",
			Usage:       "add pipelines to existing trigger",
			ArgsUsage:   "<event URI> <pipeline UID>",
			Description: "Add pipeline to existing trigger with specified event URI",
			Action:      addTriggerPipelines,
		},
		{
			Name:        "delete",
			Usage:       "delete pipeline from existing trigger",
			ArgsUsage:   "<event URI> <pipeline UID>",
			Description: "Delete pipeline from existing trigger (specified by event URI)",
			Action:      deleteTriggerPipeline,
		},
	},
}

func getTriggerPipelines(c *cli.Context) error {
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	if len(c.Args()) != 1 {
		return errors.New("wrong arguments: expected event URI")
	}
	pipelines, err := triggerReaderWriter.GetPipelines(c.Args().First())
	if err != nil {
		return err
	}
	for _, p := range pipelines {
		fmt.Println(p)
	}

	return nil
}

func addTriggerPipelines(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 2 {
		return errors.New("wrong number of arguments")
	}
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("c"), c.GlobalString("t"))
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService)
	// create pipelines
	pipelines := make([]string, 1)
	pipelines[0] = args.Get(1)
	return triggerReaderWriter.AddPipelines(args.First(), pipelines)
}

func deleteTriggerPipeline(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 2 {
		return errors.New("wrong number of arguments")
	}
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	// delete pipelines
	return triggerReaderWriter.DeletePipeline(c.Args().First(), args.Get(1))
}
