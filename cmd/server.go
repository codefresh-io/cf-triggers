package main

import (
	"fmt"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/backend"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/controller"
	"github.com/codefresh-io/hermes/pkg/version"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"
)

var serverCommand = cli.Command{
	Name: "server",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "codefresh, cf",
			Usage:  "Codefresh API endpoint",
			Value:  "https://g.codefresh.io/",
			EnvVar: "CFAPI_URL",
		},
		cli.StringFlag{
			Name:   "token, t",
			Usage:  "Codefresh API token",
			EnvVar: "CFAPI_TOKEN",
		},
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
	router := gin.Default()

	// get codefresh endpoint
	codefreshService := codefresh.NewCodefreshEndpoint(c.String("cf"), c.String("t"))

	// get trigger backend service
	triggerBackend := backend.NewRedisStore(c.GlobalString("redis"), c.GlobalInt("redis-port"), c.GlobalString("redis-password"), codefreshService)

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
	eventController := controller.NewEventController()
	router.Handle("GET", "/events/info/:id", eventController.GetEventInfo)
	router.Handle("GET", "/events/types/", eventController.ListTypes)
	router.Handle("GET", "/events/types/:type", eventController.GetType)

	// manage triggers
	router.Handle("GET", "/triggers/", triggerController.List) // pass filter or pipeline as query parameter
	router.Handle("GET", "/triggers/:id", triggerController.Get)
	router.Handle("POST", "/triggers", triggerController.Add)
	router.Handle("PUT", "/triggers/:id", triggerController.Update)
	router.Handle("DELETE", "/triggers/:id", triggerController.Delete)

	// manage pipelines attached to trigger
	router.Handle("GET", "/triggers/:id/pipelines", triggerController.GetPipelines)
	router.Handle("POST", "/triggers/:id/pipelines", triggerController.AddPipelines)
	router.Handle("DELETE", "/triggers/:id/pipelines/:pid", triggerController.DeletePipeline)

	// invoke trigger with event payload
	runnerController := controller.NewRunnerController(runner, triggerBackend, secretChecker)
	router.Handle("POST", "/trigger/:id", runnerController.TriggerEvent)

	// status handlers
	statusController := controller.NewStatusController(triggerBackend, codefreshService)
	router.GET("/health", statusController.GetHealth)
	router.GET("/version", statusController.GetVersion)
	router.GET("/ping", statusController.Ping)

	return router.Run(fmt.Sprintf(":%d", c.Int("port")))
}
