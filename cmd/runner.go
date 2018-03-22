package main

import (
	"context"
	"fmt"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
	"github.com/urfave/cli"
)

var runnerCommand = cli.Command{
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
	vars, err := util.StringSliceToMap(c.StringSlice("var"))
	if err != nil {
		return err
	}

	// get trigger pipelines for specified event (skip account check)
	ctx := context.WithValue(context.Background(), model.ContextKeyAccount, "-")
	eventURI := c.Args().First()
	pipelines, err := triggerReaderWriter.GetTriggerPipelines(ctx, eventURI, vars)
	if err != nil {
		return err
	}

	ev, err := triggerReaderWriter.GetEvent(ctx, eventURI)
	if err != nil {
		return err
	}
	account := ev.Account

	// run pipelines
	runs, err := runner.Run(account, pipelines, vars)
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
