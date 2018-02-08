package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
)

// TriggerController trigger controller
type TriggerController struct {
	runner  model.Runner
	trigger model.TriggerReaderWriter
	checker model.SecretChecker
}

// NewTriggerController new trigger controller
func NewTriggerController(runner model.Runner, trigger model.TriggerReaderWriter, checker model.SecretChecker) *TriggerController {
	return &TriggerController{runner, trigger, checker}
}

// ListEventTriggers list triggers for trigger event
func (c *TriggerController) ListEventTriggers(ctx *gin.Context) {
	// get event
	event := ctx.Params.ByName("event")
	// list trigger events, optionally filtered by type/kind and event uri filter
	if triggers, err := c.trigger.ListTriggersForEvents([]string{event}); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to list triggers for event", "error": err.Error()})
	} else {
		ctx.JSON(http.StatusOK, triggers)
	}
}

// ListPipelineTriggers list triggers for pipeline
func (c *TriggerController) ListPipelineTriggers(ctx *gin.Context) {
	// get pipeline
	pipeline := ctx.Params.ByName("pipeline")
	// list trigger events, optionally filtered by type/kind and event uri filter
	if triggers, err := c.trigger.ListTriggersForPipelines([]string{pipeline}); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to list triggers for pipeline", "error": err.Error()})
	} else {
		ctx.JSON(http.StatusOK, triggers)
	}
}

// RunTrigger pipelines for trigger
func (c *TriggerController) RunTrigger(ctx *gin.Context) {
	type RunEvent struct {
		Secret    string            `form:"secret" json:"secret" binding:"required"`
		Original  string            `form:"original" json:"original"`
		Variables map[string]string `form:"variables" json:"variables"`
	}
	// get trigger id
	event := ctx.Params.ByName("event")
	// get event payload
	var runEvent RunEvent
	if err := ctx.BindJSON(&runEvent); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "error in JSON body", "error": err.Error()})
		return
	}
	// get trigger event
	triggerEvent, err := c.trigger.GetEvent(event)
	if err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed secret validation", "error": err.Error()})
		return
	}
	if err := c.checker.Validate(runEvent.Original, runEvent.Secret, triggerEvent.Secret); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed secret validation", "error": err.Error()})
		return
	}
	// add original payload to variables
	vars := make(map[string]string)
	for k, v := range runEvent.Variables {
		vars[k] = v
	}
	vars["EVENT_PAYLOAD"] = runEvent.Original
	// get pipelines
	pipelines, err := c.trigger.GetPipelinesForTriggers([]string{event})
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
