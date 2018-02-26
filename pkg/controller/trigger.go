package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
)

// TriggerController trigger controller
type TriggerController struct {
	trigger model.TriggerReaderWriter
}

// NewTriggerController new trigger controller
func NewTriggerController(trigger model.TriggerReaderWriter) *TriggerController {
	return &TriggerController{trigger}
}

// ListEventTriggers list triggers for trigger event
func (c *TriggerController) ListEventTriggers(ctx *gin.Context) {
	// get event
	event := getParam(ctx, "event")
	// list trigger events, optionally filtered by type/kind and event uri filter
	if triggers, err := c.trigger.GetEventTriggers(getContext(ctx), event); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to list triggers for event", err.Error()})
	} else {
		ctx.JSON(http.StatusOK, triggers)
	}
}

// ListTriggers list triggers for trigger event
func (c *TriggerController) ListTriggers(ctx *gin.Context) {
	// list trigger events for all events
	if triggers, err := c.trigger.GetEventTriggers(getContext(ctx), "*"); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to list triggers for event", err.Error()})
	} else {
		ctx.JSON(http.StatusOK, triggers)
	}
}

// ListPipelineTriggers list triggers for pipeline
func (c *TriggerController) ListPipelineTriggers(ctx *gin.Context) {
	// get pipeline
	pipeline := ctx.Param("pipeline")
	// list trigger events, optionally filtered by type/kind and event uri filter
	if triggers, err := c.trigger.GetPipelineTriggers(getContext(ctx), pipeline); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to list triggers for pipeline", err.Error()})
	} else {
		ctx.JSON(http.StatusOK, triggers)
	}
}

// LinkEvent create triggers, adding multiple pipelines to the trigger event
func (c *TriggerController) LinkEvent(ctx *gin.Context) {
	// trigger event (event-uri)
	event := getParam(ctx, "event")
	// get pipelines from body
	var pipelines []string
	ctx.Bind(&pipelines)
	// perform action
	if err := c.trigger.CreateTriggersForEvent(event, pipelines); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to link trigger event to the pipelines", err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}

// UnlinkEvent delete pipeline from trigger
func (c *TriggerController) UnlinkEvent(ctx *gin.Context) {
	// get trigger event (event-uri)
	event := getParam(ctx, "event")
	// get pipeline
	pipeline := ctx.Param("pipeline")
	if err := c.trigger.DeleteTriggersForPipeline(pipeline, []string{event}); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to unlink pipeline from trigger event", err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}
