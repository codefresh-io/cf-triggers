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

// List triggers
func (c *Controller) List(ctx *gin.Context) {
	filter := ctx.Query("filter")
	pipelineUID := ctx.Query("pipeline")
	var triggers []*model.Trigger
	var err error
	// get by pipelineUID
	if pipelineUID != "" {
		if triggers, err = c.svc.ListByPipeline(pipelineUID); err != nil {
			status := http.StatusInternalServerError
			if err == model.ErrTriggerNotFound {
				status = http.StatusNotFound
			}
			ctx.JSON(status, gin.H{"status": status, "message": "failed to list triggers by pipeline", "error": err.Error()})
			return
		}
	} else {
		// get by filter
		if triggers, err = c.svc.List(filter); err != nil {
			status := http.StatusInternalServerError
			if err == model.ErrTriggerNotFound {
				status = http.StatusNotFound
			}
			ctx.JSON(status, gin.H{"status": status, "message": "failed to list triggers by filter", "error": err.Error()})
			return
		}
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

// GetPipelines get trigger pipelines
func (c *Controller) GetPipelines(ctx *gin.Context) {
	id := ctx.Params.ByName("id")
	var pipelines []string
	var err error
	if pipelines, err = c.svc.GetPipelines(id); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to get trigger pipelines", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, pipelines)
}

// AddPipelines add pipelines to trigger
func (c *Controller) AddPipelines(ctx *gin.Context) {
	// trigger id (event URI)
	id := ctx.Params.ByName("id")
	// get pipelines from body
	var pipelines []string
	ctx.Bind(&pipelines)
	// perform action
	var err error
	if err = c.svc.AddPipelines(id, pipelines); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"status": status, "message": "failed to add trigger pipelines", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, pipelines)
}

// DeletePipeline delete pipeline from trigger
func (c *Controller) DeletePipeline(ctx *gin.Context) {
	// get trigger event URI
	id := ctx.Params.ByName("id")
	// get pipeline URI
	pid := ctx.Params.ByName("pid")
	if err := c.svc.DeletePipeline(id, pid); err != nil {
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
