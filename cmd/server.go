package main

import (
	"fmt"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/controller"
	"github.com/codefresh-io/hermes/pkg/version"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var serverCommand = cli.Command{
	Name: "server",
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:  "port",
			Usage: "TCP port for the trigger manager server",
			Value: 8080,
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
	log.WithField("cfapi", c.GlobalString("codefresh")).Debug("Using Codefresh API")

	// get event handler informer
	eventHandlerInformer := backend.NewEventHandlerManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))
	log.WithField("config", c.GlobalString("config")).Debug("Monitoring types config file")

	// get trigger backend service
	triggerBackend := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService)
	log.WithFields(log.Fields{
		"redis server": c.GlobalString("redis"),
		"redis port":   c.GlobalInt("redis-port"),
	}).Debug("Using Redis backend server")

	// get pipeline runner service
	runner := backend.NewRunner(codefreshService)

	// get secret checker
	secretChecker := backend.NewSecretChecker()

	// trigger controller
	triggerController := controller.NewController(triggerBackend)

	// trigger management API
	router.Handle("GET", "/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/triggers")
	})

	// get supported events
	eventsAPI := router.Group("/events", gin.Logger())
	eventController := controller.NewEventController(triggerBackend, eventHandlerInformer)
	eventsAPI.Handle("GET", "/info/:id", eventController.GetEventInfo)
	eventsAPI.Handle("GET", "/types", eventController.ListTypes)
	eventsAPI.Handle("GET", "/types/:type/:kind", eventController.GetType)

	// manage triggers
	triggersAPI := router.Group("/triggers", gin.Logger())
	triggersAPI.Handle("GET", "/", triggerController.List) // pass filter or pipeline as query parameter
	triggersAPI.Handle("GET", "/:id", triggerController.Get)
	triggersAPI.Handle("POST", "/", triggerController.Add)
	triggersAPI.Handle("PUT", "/:id", triggerController.Update)
	triggersAPI.Handle("DELETE", "/:id", triggerController.Delete)

	// manage pipelines attached to trigger event
	triggersAPI.Handle("GET", "/:event/pipelines", triggerController.ListPipelines)
	triggersAPI.Handle("POST", "/:event/pipelines", triggerController.CreateTriggersForEvent)
	triggersAPI.Handle("DELETE", "/:event/pipelines/:pipeline", triggerController.DeleteTriggersForEvent)

	// invoke trigger with event payload
	runnerController := controller.NewRunnerController(runner, triggerBackend, secretChecker)
	triggersAPI.Handle("POST", "/:id", runnerController.TriggerEvent)

	// status handlers (without logging)
	statusController := controller.NewStatusController(triggerBackend, codefreshService)
	router.GET("/health", statusController.GetHealth)
	router.GET("/version", statusController.GetVersion)
	router.GET("/ping", statusController.Ping)

	return router.Run(fmt.Sprintf(":%d", c.Int("port")))
}
