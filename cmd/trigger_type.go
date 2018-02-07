package main

import (
	"errors"
	"fmt"

	"github.com/codefresh-io/hermes/pkg/provider"

	"github.com/urfave/cli"
)

var triggerTypeCommand = cli.Command{
	Name:  "trigger-type",
	Usage: "get information about installed event providers",
	Subcommands: []cli.Command{
		{
			Name:        "list",
			Usage:       "list trigger event provider types",
			Description: "Get all installed trigger event provider types",
			Action:      listTypes,
		},
		{
			Name:        "get",
			Usage:       "get event provider type",
			ArgsUsage:   "<type> <kind>",
			Description: "Get trigger event provider type",
			Action:      getType,
		},
	},
}

func listTypes(c *cli.Context) error {
	// get event provider informer
	eventProviderInformer := provider.NewEventProviderManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))

	types := eventProviderInformer.GetTypes()
	if types == nil {
		return errors.New("no types found")
	}

	// print types
	for _, t := range types {
		fmt.Println(t)
	}

	return nil
}

func getType(c *cli.Context) error {
	args := c.Args()
	if len(args) != 2 && len(args) != 1 {
		return errors.New("wrong number of arguments")
	}
	eventType := args.First()
	var eventKind string
	if len(args) == 2 {
		eventKind = args.Get(1)
	}

	// get event provider informer
	eventProviderInformer := provider.NewEventProviderManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))

	t, err := eventProviderInformer.GetType(eventType, eventKind)
	if err != nil {
		return err
	}

	// print type
	fmt.Println(*t)
	return nil
}
