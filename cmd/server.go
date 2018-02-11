package main

import (
	"fmt"
	"net/http"

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

	// get event provider manager
	eventProvider := provider.NewEventProviderManager(c.GlobalString("config"), c.GlobalBool("skip-monitor"))
	log.WithField("config", c.GlobalString("config")).Debug("Monitoring types config file")

	// get trigger backend service
	triggerBackend := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService, eventProvider)
	log.WithFields(log.Fields{
		"redis server": c.GlobalString("redis"),
		"redis port":   c.GlobalInt("redis-port"),
	}).Debug("Using Redis backend server")

	// get pipeline runner service
	runner := backend.NewRunner(codefreshService)

	// get secret checker
	secretChecker := backend.NewSecretChecker()

	// trigger management API
	router.Handle("GET", "/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/triggers")
	})

	// manage trigger events
	eventsAPI := router.Group("/events", gin.Logger())
	eventController := controller.NewTriggerEventController(triggerBackend)
	eventsAPI.Handle("GET", "/:event", eventController.GetEvent)
	// ?type=xxx,kind=xxx,filter=xxx
	eventsAPI.Handle("GET", "/", eventController.ListEvents)
	eventsAPI.Handle("DELETE", "/event/:event/:context", eventController.DeleteEvent)
	eventsAPI.Handle("POST", "/", eventController.CreateEvent)
	eventsAPI.Handle("POST", "/trigger/:event", eventController.LinkEvent)
	eventsAPI.Handle("DELETE", "/trigger/:event", eventController.UnlinkEvent)

	// list trigger types
	typesAPI := router.Group("/types", gin.Logger())
	typesController := controller.NewTriggerTypeController(eventProvider)
	typesAPI.Handle("GET", "/", typesController.ListTypes)
	typesAPI.Handle("GET", "/:type/:kind", typesController.GetType)

	// invoke trigger with event payload
	triggersAPI := router.Group("/triggers", gin.Logger())
	runnerController := controller.NewTriggerController(runner, triggerBackend, secretChecker)
	triggersAPI.Handle("GET", "/event/:event", runnerController.ListEventTriggers)
	triggersAPI.Handle("GET", "/pipeline/:pipeline", runnerController.ListPipelineTriggers)
	triggersAPI.Handle("POST", "/:event", runnerController.RunTrigger)

	// status handlers (without logging)
	statusController := controller.NewStatusController(triggerBackend, codefreshService)
	router.GET("/health", statusController.GetHealth)
	router.GET("/version", statusController.GetVersion)
	router.GET("/ping", statusController.Ping)

	return router.Run(fmt.Sprintf(":%d", c.Int("port")))
}
