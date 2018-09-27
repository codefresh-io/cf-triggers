package controller

import (
	"net/http"
	"strconv"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
	"encoding/json"
	"fmt"
)

// TriggerController trigger controller
type TriggerController struct {
	trigger model.TriggerReaderWriter
}

// NewTriggerController new trigger controller
func NewTriggerController(trigger model.TriggerReaderWriter) *TriggerController {
	return &TriggerController{trigger}
}

// GetEventTriggers list triggers for trigger event
func (c *TriggerController) GetEventTriggers(ctx *gin.Context) {
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
		b, _ := json.Marshal(triggers)
		fmt.Println(string(b))
		ctx.JSON(http.StatusOK, triggers)
	}
}

// GetTriggers list triggers for trigger event
func (c *TriggerController) GetTriggers(ctx *gin.Context) {
	// list trigger events for all events
	if triggers, err := c.trigger.GetEventTriggers(getContext(ctx), "*"); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to list triggers for event", err.Error()})
	} else {
		b, _ := json.Marshal(triggers)
		fmt.Println(string(b))
		ctx.JSON(http.StatusOK, triggers)
	}
}

// GetPipelineTriggers list triggers for pipeline
func (c *TriggerController) GetPipelineTriggers(ctx *gin.Context) {
	// get pipeline
	pipeline := ctx.Param("pipeline")
	// get with-event flag
	withEvent, _ := strconv.ParseBool(ctx.Query("with-event"))
	// list trigger events, optionally filtered by type/kind and event uri filter
	if triggers, err := c.trigger.GetPipelineTriggers(getContext(ctx), pipeline, withEvent); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to list triggers for pipeline", err.Error()})
	} else {
		b, _ := json.Marshal(triggers)
		fmt.Println(string(b))
		ctx.JSON(http.StatusOK, triggers)
	}
}

// CreateTrigger create triggers, adding multiple pipelines to the trigger event
func (c *TriggerController) CreateTrigger(ctx *gin.Context) {
	// trigger event (event-uri)
	event := getParam(ctx, "event")
	// get pipeline
	pipeline := ctx.Param("pipeline")
	// get request data
	type createRequest struct {
		Filters map[string]string `json:"filters,omitempty"`
	}
	// get event payload
	var request createRequest
	if err := ctx.BindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResult{http.StatusBadRequest, "error in request JSON body", err.Error()})
		return
	}
	// perform action
	if err := c.trigger.CreateTrigger(getContext(ctx), event, pipeline, request.Filters); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to create trigger: event <-> pipeline", err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}

// DeleteTrigger delete pipeline from trigger
func (c *TriggerController) DeleteTrigger(ctx *gin.Context) {
	// get trigger event (event-uri)
	event := getParam(ctx, "event")
	// get pipeline
	pipeline := ctx.Param("pipeline")
	if err := c.trigger.DeleteTrigger(getContext(ctx), event, pipeline); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to delete trigger: event <-X-> pipeline", err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}
