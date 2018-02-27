package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
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
	vars := make(map[string]string)
	for _, v := range c.StringSlice("var") {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			return fmt.Errorf("Invalid 'var' value: %s ; should be 'key=value' form", v)
		}
		vars[kv[0]] = kv[1]
	}

	// get trigger pipelines for specified event (skip account check)
	ctx := context.WithValue(context.Background(), model.ContextKeyAccount, "-")
	pipelines, err := triggerReaderWriter.GetTriggerPipelines(ctx, c.Args().First())
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
