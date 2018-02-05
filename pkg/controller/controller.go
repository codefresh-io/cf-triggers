package controller

import (
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
	svc model.TriggerReaderWriter
}

// NewController new trigger controller
func NewController(svc model.TriggerReaderWriter) *Controller {
	return &Controller{svc}
}

// Get trigger
func (c *Controller) Get(ctx *gin.Context) {
	id := ctx.Params.ByName("id")
	var trigger *model.Trigger
	var err error
	if trigger, err = c.svc.Get(id); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to get trigger", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, trigger)
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

// Add trigger
func (c *Controller) Add(ctx *gin.Context) {
	var trigger model.Trigger
	ctx.Bind(&trigger)

	if trigger.Event != "" && len(trigger.Pipelines) != 0 {
		// add trigger
		if err := c.svc.Add(trigger); err != nil {
			status := http.StatusInternalServerError
			if err == model.ErrTriggerAlreadyExists {
				status = http.StatusBadRequest
			}
			ctx.JSON(status, gin.H{"status": status, "message": "failed to add trigger", "error": err.Error()})
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
	var trigger model.Trigger
	ctx.Bind(&trigger)

	if trigger.Event != "" && len(trigger.Pipelines) != 0 {
		// update trigger
		if err := c.svc.Update(trigger); err != nil {
			status := http.StatusInternalServerError
			if err == model.ErrTriggerNotFound {
				status = http.StatusNotFound
			}
			ctx.JSON(status, gin.H{"status": status, "message": "failed to update trigger", "error": err.Error()})
			return
		}
		// report OK
		ctx.Status(http.StatusOK)
	} else {
		// Display error
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"status": http.StatusUnprocessableEntity, "message": "required fields are empty"})
	}
}

// Delete trigger
func (c *Controller) Delete(ctx *gin.Context) {
	id := ctx.Params.ByName("id")
	if err := c.svc.Delete(id); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to delete trigger", "error": err.Error()})
		return
	}
	ctx.Status(http.StatusOK)
}
