package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/provider"
	"github.com/gin-gonic/gin"
)

// TriggerTypeController trigger controller
type TriggerTypeController struct {
	eventProvider provider.EventProvider
}

// NewTriggerTypeController new trigger controller
func NewTriggerTypeController(eventProvider provider.EventProvider) *TriggerTypeController {
	return &TriggerTypeController{eventProvider}
}

// ListTypes get registered trigger types
func (c *TriggerTypeController) ListTypes(ctx *gin.Context) {
	types := c.eventProvider.GetTypes()
	if types == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "no types found"})
		return
	}
	ctx.JSON(http.StatusOK, types)
}

// GetType get details for specific trigger type
func (c *TriggerTypeController) GetType(ctx *gin.Context) {
	// get event type and kind
	eventType := ctx.Param("type")
	if eventType == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "missing event type"})
		return
	}
	// get event kind
	eventKind := ctx.Param("kind")
	typeObject, err := c.eventProvider.GetType(eventType, eventKind)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "failed to find type " + eventType + " of kind " + eventKind, "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, typeObject)
}
