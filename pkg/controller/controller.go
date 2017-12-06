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
	Variables map[string]string `form:"variables" json:"variables" binding:"required"`
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Failed to get list of triggers!"})
		return
	}
	if len(triggers) <= 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No triggers found!"})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Failed to get trigger!"})
		return
	}
	if trigger.IsEmpty() {
		ctx.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": fmt.Sprintf("No trigger %s found!", id)})
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
			ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Failed to add trigger!"})
			return
		}
		// report OK
		ctx.Status(http.StatusOK)
	} else {
		// Display error
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"status": http.StatusUnprocessableEntity, "message": "Required fields are empty!"})
	}
}

// Update trigger
func (c *Controller) Update(ctx *gin.Context) {

}

// Delete trigger
func (c *Controller) Delete(ctx *gin.Context) {
	id := ctx.Params.ByName("id")
	if err := c.svc.Delete(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Failed to delete trigger!"})
		return
	}
	ctx.Status(http.StatusOK)
}

// Run pipelines for trigger
func (c *Controller) Run(ctx *gin.Context) {
	// get trigger id
	id := ctx.Params.ByName("id")
	// get event payload
	var event Event
	if err := ctx.BindJSON(&event); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": err.Error()})
		return
	}
	// check secret
	if err := c.svc.CheckSecret(id, event.Secret); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Invalid secret!"})
		return
	}
	// run pipelines
	if err := c.svc.Run(id, event.Variables); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Failed to run trigger pipelines!"})
		return
	}
	ctx.Status(http.StatusOK)
}
