package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
)

// Controller trigger controller
type Controller struct {
	svc model.TriggerReaderWriter
}

// NewController new trigger controller
func NewController(svc model.TriggerReaderWriter) *Controller {
	return &Controller{svc}
}

// GetEvent get trigger event
func (c *Controller) GetEvent(ctx *gin.Context) {
	event := ctx.Params.ByName("event")
	var triggerEvent *model.Event
	var err error
	if triggerEvent, err = c.svc.GetEvent(event); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to get trigger", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, triggerEvent)
}

// ListPipelines get trigger pipelines
func (c *Controller) ListPipelines(ctx *gin.Context) {
	event := ctx.Params.ByName("event")
	var pipelines []string
	var err error
	if pipelines, err = c.svc.GetPipelinesForTriggers([]string{event}); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to get list pipelines", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, pipelines)
}

// CreateTriggersForEvent create triggers, adding multiple pipelines to the trigger event
func (c *Controller) CreateTriggersForEvent(ctx *gin.Context) {
	// trigger event (event-uri)
	event := ctx.Params.ByName("event")
	// get pipelines from body
	var pipelines []string
	ctx.Bind(&pipelines)
	// perform action
	var err error
	if err = c.svc.CreateTriggersForEvent(event, pipelines); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to create trigger for pipelines", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, pipelines)
}

// DeleteTriggersForEvent delete pipeline from trigger
func (c *Controller) DeleteTriggersForEvent(ctx *gin.Context) {
	// get trigger event (event-uri)
	event := ctx.Params.ByName("event")
	// get pipeline
	pipeline := ctx.Params.ByName("pipeline")
	if err := c.svc.DeleteTriggersForPipeline(pipeline, []string{event}); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to remove pipeline from trigger", "error": err.Error()})
		return
	}
	ctx.Status(http.StatusOK)
}

// CreateEvent create trigger event
func (c *Controller) CreateEvent(ctx *gin.Context) {
	type createReq struct {
		Type   string            `json:"type"`
		Kind   string            `json:"kind"`
		Secret string            `json:"secret,omitempty"`
		Values map[string]string `json:"values"`
	}
	var req createReq
	ctx.Bind(&req)

	// create trigger event
	if event, err := c.svc.CreateEvent(req.Type, req.Kind, req.Secret, req.Values); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerAlreadyExists {
			status = http.StatusBadRequest
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to add trigger", "error": err.Error()})
	} else {
		// report OK and event URI
		ctx.JSON(http.StatusOK, event.URI)
	}
}

// DeleteEvent delete trigger event
func (c *Controller) DeleteEvent(ctx *gin.Context) {
	event := ctx.Params.ByName("event")
	if err := c.svc.DeleteEvent(event); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to delete trigger", "error": err.Error()})
		return
	}
	ctx.Status(http.StatusOK)
}
