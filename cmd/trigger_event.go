package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/provider"

	"github.com/urfave/cli"
)

var triggerEventCommand = cli.Command{
	Name:  "trigger-event",
	Usage: "manage trigger events",
	Subcommands: []cli.Command{
		{
			Name: "list",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "type",
					Usage: "trigger event type",
				},
				cli.StringFlag{
					Name:  "kind",
					Usage: "trigger event kind",
				},
				cli.StringFlag{
					Name:  "filter",
					Usage: "event URI filter",
				},
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
				},
				cli.BoolFlag{
					Name:  "quiet,q",
					Usage: "only display trigger event URIs",
				},
			},
			Usage:       "list defined trigger events",
			Description: "List trigger events",
			Action:      listEvents,
		},
		{
			Name:        "get",
			Usage:       "get trigger event by event URI",
			ArgsUsage:   "<event-uri>",
			Description: "Get single trigger event",
			Action:      getEvent,
		},
		{
			Name: "create",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "type",
					Usage: "trigger event type",
				},
				cli.StringFlag{
					Name:  "kind",
					Usage: "trigger event kind",
				},
				cli.StringFlag{
					Name:  "secret",
					Usage: "trigger event secret (auto-generated when skipped)",
					Value: "!generate",
				},
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
				},
				cli.StringSliceFlag{
					Name:  "value",
					Usage: "trigger event values (key=value pairs); as defined by trigger type config",
				},
				cli.StringSliceFlag{
					Name:  "context",
					Usage: "Codefresh context with required credentials",
				},
			},
			Usage:       "create trigger event",
			Description: "Create/define trigger event",
			Action:      createEvent,
		},
		{
			Name: "delete",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "type",
					Usage: "trigger event type",
				},
				cli.StringFlag{
					Name:  "kind",
					Usage: "trigger event kind",
				},
				cli.StringFlag{
					Name:  "context",
					Usage: "Codefresh context with required credentials",
				},
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
				},
			},
			Usage:       "delete trigger event",
			ArgsUsage:   "<event-uri>",
			Description: "Delete/undefine trigger event by event URI",
			Action:      deleteEvent,
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

func listEvents(c *cli.Context) error {
	// get trigger backend
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get trigger events
	events, err := triggerReaderWriter.GetEvents(c.String("type"), c.String("kind"), c.String("filter"), c.String("account"))
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return errors.New("no trigger events found")
	}
	for _, event := range events {
		if c.Bool("quiet") {
			fmt.Println(event.URI)
		} else {
			fmt.Println(event)
		}
	}
	return nil
}

func getEvent(c *cli.Context) error {
	// get trigger backend
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get trigger events
	event, err := triggerReaderWriter.GetEvent(c.Args().First())
	if err != nil {
		return err
	}
	fmt.Println(event)
	return nil
}

func createEvent(c *cli.Context) error {
	// get event provider informer
	eventProvider := provider.NewEventProviderManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))
	// get trigger backend
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, eventProvider)
	// construct values map
	values := make(map[string]string)
	valueFlag := c.StringSlice("value")
	for _, v := range valueFlag {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			return errors.New("Invalid value format, must be in form '--value key=val'")
		}
		values[kv[0]] = kv[1]
	}
	// create new event
	event, err := triggerReaderWriter.CreateEvent(c.String("type"), c.String("kind"), c.String("secret"), c.String("account"), c.String("context"), values)
	if err != nil {
		return err
	}
	// print it out
	fmt.Println("New trigger event successfully created.")
	fmt.Println(event.URI)

	return nil
}

func deleteEvent(c *cli.Context) error {
	// get trigger backend
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get trigger events
	err := triggerReaderWriter.DeleteEvent(c.Args().First(), c.String("context"), c.String("account"))
	if err != nil {
		return err
	}
	fmt.Println("Trigger event successfully deleted.")
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
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService, nil)
	// create triggers for event linking it to passed pipeline(s)
	return triggerReaderWriter.CreateTriggersForEvent(c.Args().First(), c.Args().Tail())
}

func unlinkEvent(c *cli.Context) error {
	// get trigger name and pipeline
	args := c.Args()
	if len(args) != 2 {
		return errors.New("wrong number of arguments")
	}
	// get trigger service
	triggerReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// delete pipelines
	return triggerReaderWriter.DeleteTriggersForEvent(args.First(), args.Tail())
}
