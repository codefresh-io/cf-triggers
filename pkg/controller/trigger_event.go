package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
)

// TriggerEventController trigger controller
type TriggerEventController struct {
	svc model.TriggerReaderWriter
}

// NewTriggerEventController new trigger controller
func NewTriggerEventController(svc model.TriggerReaderWriter) *TriggerEventController {
	return &TriggerEventController{svc}
}

// ListEvents get defined trigger events
func (c *TriggerEventController) ListEvents(ctx *gin.Context) {
	eventType := ctx.Params.ByName("type")
	kind := ctx.Params.ByName("kind")
	filter := ctx.Params.ByName("filter")
	// list trigger events, optionally filtered by type/kind and event uri filter
	if events, err := c.svc.GetEvents(eventType, kind, filter); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to list trigger events", "error": err.Error()})
	} else {
		ctx.JSON(http.StatusOK, events)
	}
}

// GetEvent get trigger event
func (c *TriggerEventController) GetEvent(ctx *gin.Context) {
	event := ctx.Params.ByName("event")
	if triggerEvent, err := c.svc.GetEvent(event); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to get trigger event", "error": err.Error()})
	} else {
		ctx.JSON(http.StatusOK, triggerEvent)
	}
}

// CreateEvent create trigger event
func (c *TriggerEventController) CreateEvent(ctx *gin.Context) {
	type createReq struct {
		Type        string            `json:"type"`
		Kind        string            `json:"kind"`
		Secret      string            `json:"secret,omitempty"`
		Credentials map[string]string `json:"credentials,omitempty"`
		Values      map[string]string `json:"values"`
	}
	var req createReq
	ctx.Bind(&req)

	// create trigger event
	if event, err := c.svc.CreateEvent(req.Type, req.Kind, req.Secret, req.Credentials, req.Values); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerAlreadyExists {
			status = http.StatusBadRequest
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to add trigger event", "error": err.Error()})
	} else {
		// report OK and event URI
		ctx.JSON(http.StatusOK, event.URI)
	}
}

// DeleteEvent delete trigger event
func (c *TriggerEventController) DeleteEvent(ctx *gin.Context) {
	event := ctx.Params.ByName("event")
	// get credentials from DELETE body
	type deleteReq struct {
		Credentials map[string]string `json:"credentials,omitempty"`
	}
	var req deleteReq
	ctx.Bind(&req)

	if err := c.svc.DeleteEvent(event, req.Credentials); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to delete trigger event", "error": err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}

// LinkEvent create triggers, adding multiple pipelines to the trigger event
func (c *TriggerEventController) LinkEvent(ctx *gin.Context) {
	// trigger event (event-uri)
	event := ctx.Params.ByName("event")
	// get pipelines from body
	var pipelines []string
	ctx.Bind(&pipelines)
	// perform action
	if err := c.svc.CreateTriggersForEvent(event, pipelines); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to link trigger event to the pipelines", "error": err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}

// UnlinkEvent delete pipeline from trigger
func (c *TriggerEventController) UnlinkEvent(ctx *gin.Context) {
	// get trigger event (event-uri)
	event := ctx.Params.ByName("event")
	// get pipeline
	pipeline := ctx.Params.ByName("pipeline")
	if err := c.svc.DeleteTriggersForPipeline(pipeline, []string{event}); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to unlink pipeline from trigger event", "error": err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}
