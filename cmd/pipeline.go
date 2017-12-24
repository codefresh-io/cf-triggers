package main

import (
	"errors"
	"fmt"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	log "github.com/sirupsen/logrus"
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
			ArgsUsage:   "<event URI> <pipeline repo-owner> <pipeline repo-name> <pipeline name>",
			Description: "Add pipeline to existing trigger with specified event URI",
			Action:      addTriggerPipelines,
		},
		{
			Name:        "delete",
			Usage:       "delete pipeline from existing trigger",
			ArgsUsage:   "<event URI> <pipeline repo-owner> <pipeline repo-name> <pipeline name>",
			Description: "Delete pipeline from existing trigger with specified event URI",
			Action:      deleteTriggerPipeline,
		},
	},
}

func getTriggerPipelines(c *cli.Context) error {
	triggerService := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	if len(c.Args()) != 1 {
		return errors.New("wrong arguments: expected event URI")
	}
	pipelines, err := triggerService.GetPipelines(c.Args().First())
	if err != nil {
		log.Error(err)
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
	if len(args) != 4 {
		return errors.New("wrong number of arguments")
	}
	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("cf"), c.GlobalString("t"))
	// get trigger service
	triggerService := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService)
	// create pipelines
	pipelines := make([]model.Pipeline, 1)
	pipelines[0] = model.Pipeline{RepoOwner: args.Get(1), RepoName: args.Get(2), Name: args.Get(3)}
	return triggerService.AddPipelines(args.First(), pipelines)
}

func deleteTriggerPipeline(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 4 {
		return errors.New("wrong number of arguments")
	}
	// get trigger service
	triggerService := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	// create pipelines
	pid := fmt.Sprintf("%s:%s:%s", args.Get(1), args.Get(2), args.Get(3))
	return triggerService.DeletePipeline(c.Args().First(), pid)
}
