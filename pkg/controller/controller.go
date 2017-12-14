package controller

import (
	"fmt"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
)

// Event binding from JSON
type Event struct {
	Secret    string            `form:"secret" json:"secret" binding:"required"`
	Original  string            `form:"original" json:"original"`
	Variables map[string]string `form:"variables" json:"variables"`
}

// Controller trigger controller
type Controller struct {
	svc model.TriggerService
}

// NewController new trigger controller
func NewController(svc model.TriggerService) *Controller {
	return &Controller{svc}
}

// List triggers
func (c *Controller) List(ctx *gin.Context) {
	filter := ctx.Query("filter")
	var triggers []model.Trigger
	var err error
	if triggers, err = c.svc.List(filter); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to list triggers", "error": err.Error()})
		return
	}
	if len(triggers) <= 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "no triggers found"})
		return
	}
	ctx.JSON(http.StatusOK, triggers)
}

// Get trigger
func (c *Controller) Get(ctx *gin.Context) {
	id := ctx.Params.ByName("id")
	var trigger model.Trigger
	var err error
	if trigger, err = c.svc.Get(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to get trigger", "error": err.Error()})
		return
	}
	if trigger.IsEmpty() {
		ctx.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": fmt.Sprintf("trigger %s not found", id)})
		return
	}
	ctx.JSON(http.StatusOK, trigger)
}

// Add trigger
func (c *Controller) Add(ctx *gin.Context) {
	var trigger model.Trigger
	ctx.Bind(&trigger)

	if trigger.Event != "" && len(trigger.Pipelines) != 0 {
		// add trigger
		if err := c.svc.Add(trigger); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to add trigger", "error": err.Error()})
			return
		}
		// report OK
		ctx.Status(http.StatusOK)
	} else {
		// Display error
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"status": http.StatusUnprocessableEntity, "message": "required fields are empty"})
	}
}

// Update trigger
func (c *Controller) Update(ctx *gin.Context) {

}

// Delete trigger
func (c *Controller) Delete(ctx *gin.Context) {
	id := ctx.Params.ByName("id")
	if err := c.svc.Delete(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to delete trigger", "error": err.Error()})
		return
	}
	ctx.Status(http.StatusOK)
}

// TriggerEvent pipelines for trigger
func (c *Controller) TriggerEvent(ctx *gin.Context) {
	// get trigger id
	id := ctx.Params.ByName("id")
	// get event payload
	var event Event
	if err := ctx.BindJSON(&event); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "error in JSON body", "error": err.Error()})
		return
	}
	// check secret
	if err := c.svc.CheckSecret(id, event.Original, event.Secret); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed secret validation", "error": err.Error()})
		return
	}
	// add original payload to variables
	vars := make(map[string]string)
	for k, v := range event.Variables {
		vars[k] = v
	}
	vars["EVENT_PAYLOAD"] = event.Original
	// run pipelines
	runs, err := c.svc.Run(id, vars)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to run trigger pipelines", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, runs)
}

// GetHealth status
func (c *Controller) GetHealth(ctx *gin.Context) {
	_, err := c.svc.Ping()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to talk to Redis", "error": err.Error()})
	} else {
		ctx.String(http.StatusOK, "Healthy")
	}
}
