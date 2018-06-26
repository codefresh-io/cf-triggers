package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/model"
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
					Value: model.PublicAccount,
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
			Name: "get",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
					Value: model.PublicAccount,
				},
			},
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
					Value: model.PublicAccount,
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
					Name:  "header",
					Usage: "header name for signature validation",
				},
				cli.StringFlag{
					Name:  "account",
					Usage: "Codefresh account ID",
					Value: model.PublicAccount,
				},
			},
			Usage:       "delete trigger event",
			ArgsUsage:   "<event-uri>",
			Description: "Delete/undefine trigger event by event URI",
			Action:      deleteEvent,
		},
	},
}

func getContext(c *cli.Context) context.Context {
	account := c.String("account")
	return context.WithValue(context.Background(), model.ContextKeyAccount, account)
}

func listEvents(c *cli.Context) error {
	// get trigger backend
	eventReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get trigger events
	events, err := eventReaderWriter.GetEvents(getContext(c), c.String("type"), c.String("kind"), c.String("filter"))
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
	eventReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get trigger events
	event, err := eventReaderWriter.GetEvent(getContext(c), c.Args().First())
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
	eventReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, eventProvider)
	// construct values map
	values := make(map[string]string)
	valueFlag := c.StringSlice("value")
	for _, v := range valueFlag {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			return errors.New("invalid value format, must be in form '--value key=val'")
		}
		values[kv[0]] = kv[1]
	}
	// create new event
	event, err := eventReaderWriter.CreateEvent(getContext(c), c.String("type"), c.String("kind"), c.String("secret"), c.String("context"), c.String("header"), values)
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
	eventReaderWriter := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), nil, nil)
	// get trigger events
	err := eventReaderWriter.DeleteEvent(getContext(c), c.Args().First(), c.String("context"))
	if err != nil {
		return err
	}
	fmt.Println("Trigger event successfully deleted.")
	return nil
}
