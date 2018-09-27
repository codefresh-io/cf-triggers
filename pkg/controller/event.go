package controller

import (
	"context"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
	"fmt"
)

// TriggerEventController trigger controller
type TriggerEventController struct {
	svc model.TriggerEventReaderWriter
}

// NewTriggerEventController new trigger controller
func NewTriggerEventController(svc model.TriggerEventReaderWriter) *TriggerEventController {
	return &TriggerEventController{svc}
}

// GetEvents get defined trigger events
func (c *TriggerEventController) GetEvents(ctx *gin.Context) {
	eventType := ctx.Query("type")
	kind := ctx.Query("kind")
	filter := ctx.Query("filter")

	// for public event create new context
	actionContext := getContext(ctx)
	if ctx.Query("public") == "true" {
		actionContext = context.WithValue(actionContext, model.ContextKeyPublic, true)
	}

	// list trigger events, optionally filtered by type/kind and event uri filter
	if events, err := c.svc.GetEvents(actionContext, eventType, kind, filter); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to list trigger events", err.Error()})
	} else {
		ctx.JSON(http.StatusOK, events)
	}
}

// GetEvent get trigger event
func (c *TriggerEventController) GetEvent(ctx *gin.Context) {
	event := getParam(ctx, "event")
	if triggerEvent, err := c.svc.GetEvent(getContext(ctx), event); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to get trigger event", err.Error()})
	} else {
		ctx.JSON(http.StatusOK, triggerEvent)
	}
}

// CreateEvent create trigger event
func (c *TriggerEventController) CreateEvent(ctx *gin.Context) {
	type createReq struct {
		Type    string            `json:"type"`
		Kind    string            `json:"kind"`
		Secret  string            `json:"secret,omitempty"`
		Context string            `json:"context,omitempty"`
		Values  map[string]string `json:"values"`
	}
	var req createReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResult{http.StatusBadRequest, "error in request JSON body", err.Error()})
		return
	}

	fmt.Println("CREATE TRIGGER SECRET " + req.Secret)

	// for public event create new context
	actionContext := getContext(ctx)
	if ctx.Query("public") == "true" {
		actionContext = context.WithValue(actionContext, model.ContextKeyPublic, true)
	}

	// create trigger event
	if event, err := c.svc.CreateEvent(actionContext, req.Type, req.Kind, req.Secret, req.Context, req.Values); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerAlreadyExists {
			status = http.StatusBadRequest
		}
		ctx.JSON(status, ErrorResult{status, "failed to add trigger event", err.Error()})
	} else {
		// report OK and event URI
		ctx.JSON(http.StatusOK, event.URI)
	}
}

// DeleteEvent delete trigger event
func (c *TriggerEventController) DeleteEvent(ctx *gin.Context) {
	event := getParam(ctx, "event")
	context := ctx.Params.ByName("context")

	if err := c.svc.DeleteEvent(getContext(ctx), event, context); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to delete trigger event", err.Error()})
	} else {
		ctx.Status(http.StatusOK)
	}
}
