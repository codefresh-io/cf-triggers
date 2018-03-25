package controller

import (
	"context"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

// RunnerController trigger controller
type RunnerController struct {
	runnerSvc  model.Runner
	eventSvc   model.TriggerEventReaderWriter
	triggerSvc model.TriggerReaderWriter
	checkerSvc model.SecretChecker
}

// NewRunnerController new runner controller
func NewRunnerController(runnerSvc model.Runner, eventSvc model.TriggerEventReaderWriter, triggerSvc model.TriggerReaderWriter, checkerSvc model.SecretChecker) *RunnerController {
	return &RunnerController{
		runnerSvc:  runnerSvc,
		eventSvc:   eventSvc,
		triggerSvc: triggerSvc,
		checkerSvc: checkerSvc}
}

// RunTrigger pipelines for trigger
func (c *RunnerController) RunTrigger(ctx *gin.Context) {
	type RunEvent struct {
		Secret    string            `form:"secret" json:"secret" binding:"required"`
		Original  string            `form:"original" json:"original"`
		Variables map[string]string `form:"variables" json:"variables"`
	}
	// get trigger event
	event := getParam(ctx, "event")
	log.WithField("event", event).Debug("triggering pipelines for event")
	// get event payload
	var runEvent RunEvent
	if err := ctx.BindJSON(&runEvent); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResult{http.StatusBadRequest, "error in JSON body", err.Error()})
		return
	}
	// prepare context for specified event (skip account check)
	allCtx := context.WithValue(context.Background(), model.ContextKeyAccount, "-")
	// get trigger event
	triggerEvent, err := c.eventSvc.GetEvent(allCtx, event)
	if err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to get event", err.Error()})
		return
	}
	if err := c.checkerSvc.Validate(runEvent.Original, runEvent.Secret, triggerEvent.Secret); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed secret validation", err.Error()})
		return
	}
	// add original payload to variables
	vars := make(map[string]string)
	for k, v := range runEvent.Variables {
		vars[k] = v
	}
	vars["EVENT_PAYLOAD"] = runEvent.Original
	// get connected pipelines
	pipelines, err := c.triggerSvc.GetTriggerPipelines(allCtx, event, vars)
	if err != nil {
		// if there are no pipelines connected to the trigger event don't fail this REST method
		// to avoid multiple 'errors' reported to the event provider log
		// it's possible to have trigger event defined and not connected to any pipeline
		if err == model.ErrPipelineNotFound || err == model.ErrTriggerNotFound {
			log.WithField("event", event).Debug("there are no pipelines associated with trigger event")
			ctx.Status(http.StatusNoContent)
			return
		}
		ctx.JSON(http.StatusInternalServerError, ErrorResult{http.StatusInternalServerError, "failed to run trigger pipelines", err.Error()})
		return
	}
	// run pipelines
	runs, err := c.runnerSvc.Run(triggerEvent.Account, pipelines, vars)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResult{http.StatusInternalServerError, "failed to run trigger pipelines", err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, runs)
}
