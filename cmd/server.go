package main

import (
	"fmt"

	"github.com/codefresh-io/hermes/pkg/model"

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

func setupRouter(eventReaderWriter model.TriggerEventReaderWriter,
	triggerReaderWriter model.TriggerReaderWriter,
	eventProvider provider.EventProvider,
	runner model.Runner,
	checker model.SecretChecker,
	pinger model.Pinger,
	pipelineService codefresh.PipelineService) *gin.Engine {
	// Creates a router without any middleware by default
	router := gin.New()
	router.Use(gin.Recovery())

	// manage trigger events
	eventController := controller.NewTriggerEventController(eventReaderWriter)
	eventsAPI := router.Group("/account/:account/events", gin.Logger())
	{
		eventsAPI.Handle("GET", "/", eventController.GetEvents)
		eventsAPI.Handle("GET", "/:event", eventController.GetEvent)
		eventsAPI.Handle("DELETE", "/:event/*context", eventController.DeleteEvent)
		eventsAPI.Handle("POST", "/", eventController.CreateEvent)
	}

	// manage triggers
	triggerController := controller.NewTriggerController(triggerReaderWriter)
	triggersAPI := router.Group("/accounts/:account/triggers", gin.Logger())
	{
		triggersAPI.Handle("GET", "/", triggerController.GetTriggers)
		triggersAPI.Handle("GET", "/event/:event", triggerController.GetEventTriggers)
		triggersAPI.Handle("GET", "/pipeline/:pipeline", triggerController.GetPipelineTriggers)
		triggersAPI.Handle("POST", "/:event/:pipeline", triggerController.CreateTrigger)
		triggersAPI.Handle("DELETE", "/:event/:pipeline", triggerController.DeleteTrigger)
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
	runnerController := controller.NewRunnerController(runner, eventReaderWriter, triggerReaderWriter, checker)
	{
		runAPI.Handle("POST", "/:event", runnerController.RunTrigger)
	}

	// status handlers (without logging)
	statusController := controller.NewStatusController(pinger, pipelineService)
	{
		router.GET("/health", statusController.GetHealth)
		router.GET("/version", statusController.GetVersion)
		router.GET("/ping", statusController.Ping)
	}

	return router
}

// start trigger manager server
func runServer(c *cli.Context) error {
	fmt.Println()
	fmt.Println(version.ASCIILogo)

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
	checker := backend.NewSecretChecker()

	// setup router
	router := setupRouter(triggerBackend, triggerBackend, eventProvider, runner, checker, triggerBackend, codefreshService)

	// start router
	port := c.Int("port")
	log.WithField("port", port).Debug("starting hermes server")
	return router.Run(fmt.Sprintf(":%d", port))
}
