package main

import (
	"fmt"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/controller"
	"github.com/codefresh-io/hermes/pkg/provider"
	"github.com/codefresh-io/hermes/pkg/version"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var serverCommand = cli.Command{
	Name: "server",
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:   "port",
			Usage:  "TCP port for the trigger manager server",
			Value:  9011,
			EnvVar: "PORT",
		},
	},
	Usage:       "start trigger manager server",
	Description: "Run Codefresh trigger manager server. Use REST API to manage triggers. Send normalized event payload to trigger endpoint to invoke associated Codefresh pipelines.",
	Action:      runServer,
}

// start trigger manager server
func runServer(c *cli.Context) error {
	fmt.Println()
	fmt.Println(version.ASCIILogo)

	// Creates a router without any middleware by default
	router := gin.New()
	router.Use(gin.Recovery())

	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.GlobalString("codefresh"), c.GlobalString("token"))
	log.WithField("cfapi", c.GlobalString("codefresh")).Debug("using Codefresh API")

	// get event provider manager
	eventProvider := provider.NewEventProviderManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))
	log.WithField("config", c.GlobalString("config")).Debug("monitoring types config file")

	// get trigger backend service
	triggerBackend := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService, eventProvider)
	log.WithFields(log.Fields{
		"redis server": c.GlobalString("redis"),
		"redis port":   c.GlobalInt("redis-port"),
	}).Debug("using Redis backend server")

	// get pipeline runner service
	runner := backend.NewRunner(codefreshService)

	// get secret checker
	secretChecker := backend.NewSecretChecker()

	// manage trigger events
	eventController := controller.NewTriggerEventController(triggerBackend)
	for _, route := range []string{"/events", "/account/:account/events"} {
		eventsAPI := router.Group(route, gin.Logger())
		{
			eventsAPI.Handle("GET", "/", eventController.ListEvents)
			eventsAPI.Handle("GET", "/:event", eventController.GetEvent)
			eventsAPI.Handle("DELETE", "/event/:event/*context", eventController.DeleteEvent)
			eventsAPI.Handle("POST", "/", eventController.CreateEvent)
		}
	}

	// manage triggers
	triggerController := controller.NewTriggerController(triggerBackend)
	for _, route := range []string{"/triggers", "/accounts/:account/triggers"} {
		triggersAPI := router.Group(route, gin.Logger())
		{
			triggersAPI.Handle("GET", "/", triggerController.ListTriggers)
			triggersAPI.Handle("GET", "/:event", triggerController.ListEventTriggers)
			triggersAPI.Handle("GET", "/pipeline/:pipeline", triggerController.ListPipelineTriggers)
			triggersAPI.Handle("POST", "/:event", triggerController.LinkEvent)
			triggersAPI.Handle("DELETE", "/:event", triggerController.UnlinkEvent)
		}
	}

	// list trigger types
	typesController := controller.NewTriggerTypeController(eventProvider)
	typesAPI := router.Group("/types", gin.Logger())
	{
		typesAPI.Handle("GET", "/", typesController.ListTypes)
		typesAPI.Handle("GET", "/:type/:kind", typesController.GetType)
	}

	// invoke trigger with event payload
	runAPI := router.Group("/run", gin.Logger())
	runnerController := controller.NewRunnerController(runner, triggerBackend, triggerBackend, secretChecker)
	{
		runAPI.Handle("POST", "/:event", runnerController.RunTrigger)
	}

	// status handlers (without logging)
	statusController := controller.NewStatusController(triggerBackend, codefreshService)
	{
		router.GET("/health", statusController.GetHealth)
		router.GET("/version", statusController.GetVersion)
		router.GET("/ping", statusController.Ping)
	}

	port := c.Int("port")
	log.WithField("port", port).Debug("starting hermes server")
	return router.Run(fmt.Sprintf(":%d", port))
}
