package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
)

// RunnerController trigger controller
type RunnerController struct {
	runner  model.Runner
	trigger model.TriggerReaderWriter
	checker model.SecretChecker
}

// NewRunnerController new trigger controller
func NewRunnerController(runner model.Runner, trigger model.TriggerReaderWriter, checker model.SecretChecker) *RunnerController {
	return &RunnerController{runner, trigger, checker}
}

// TriggerEvent pipelines for trigger
func (c *RunnerController) TriggerEvent(ctx *gin.Context) {
	// get trigger id
	id := ctx.Params.ByName("id")
	// get event payload
	var event Event
	if err := ctx.BindJSON(&event); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "error in JSON body", "error": err.Error()})
		return
	}
	// get trigger
	trigger, err := c.trigger.Get(id)
	if err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed secret validation", "error": err.Error()})
		return
	}
	if err := c.checker.Validate(event.Original, event.Secret, trigger.Secret); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed secret validation", "error": err.Error()})
		return
	}
	// add original payload to variables
	vars := make(map[string]string)
	for k, v := range event.Variables {
		vars[k] = v
	}
	vars["EVENT_PAYLOAD"] = event.Original
	// get pipelines
	pipelines, err := c.trigger.GetPipelines(id)
	if err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrPipelineNotFound || err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to run trigger pipelines", "error": err.Error()})
		return
	}
	// run pipelines
	runs, err := c.runner.Run(pipelines, vars)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to run trigger pipelines", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, runs)
}
