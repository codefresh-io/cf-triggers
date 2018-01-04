package main

import (
	"errors"
	"fmt"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/urfave/cli"
)

var infoCommand = cli.Command{
	Name:  "info",
	Usage: "get information about installed event handlers and events",
	Subcommands: []cli.Command{
		{
			Name:        "list",
			Usage:       "list event handler types",
			Description: "Get all installed Event Handler types",
			Action:      listTypes,
		},
		{
			Name:        "get",
			Usage:       "get event handler type",
			ArgsUsage:   "<type> <kind>",
			Description: "Get event handler type, specified by event type and (optionally) kind",
			Action:      getType,
		},
		{
			Name:        "event",
			Usage:       "get event details",
			ArgsUsage:   "<event URI>",
			Description: "Get event details from event uri",
			Action:      getEventInfo,
		},
	},
}

func listTypes(c *cli.Context) error {
	// get event handler informer
	eventHandlerInformer := backend.NewEventHandlerManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))

	types := eventHandlerInformer.GetTypes()
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

	// get event handler informer
	eventHandlerInformer := backend.NewEventHandlerManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))

	t := eventHandlerInformer.GetType(eventType, eventKind)
	if t == nil {
		return fmt.Errorf("type %s not found", eventType)
	}

	// print type
	fmt.Println(*t)
	return nil
}

func getEventInfo(c *cli.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return errors.New("wrong number of arguments")
	}
	eventURI := args.First()

	// get event secret from trigger backend
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil)
	secret, err := triggerReaderWriter.GetSecret(eventURI)
	if err != nil {
		return err
	}

	// get event handler informer
	eventHandlerInformer := backend.NewEventHandlerManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))

	info := eventHandlerInformer.GetEventInfo(eventURI, secret)
	if info == nil {
		return fmt.Errorf("event info for %s not found", eventURI)
	}

	// print event info
	fmt.Println(*info)

	return nil
}
