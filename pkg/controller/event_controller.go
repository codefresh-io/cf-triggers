package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// EventController trigger controller
type EventController struct{}

// NewEventController new trigger controller
func NewEventController() *EventController {
	return &EventController{}
}

// ListTypes get registered trigger types
func (c *EventController) ListTypes(ctx *gin.Context) {
	ctx.String(http.StatusNotImplemented, "Not implemented")
}

// GetType get details for specific trigger type
func (c *EventController) GetType(ctx *gin.Context) {
	ctx.String(http.StatusNotImplemented, "Not implemented")
}

// GetEventInfo get human readable text for event (ask Event Provider)
func (c *EventController) GetEventInfo(ctx *gin.Context) {
	ctx.String(http.StatusNotImplemented, "Not implemented")
}
