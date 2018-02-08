package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
)

// PipelineController trigger controller
type PipelineController struct {
	svc model.TriggerReaderWriter
}

// NewPipelineController new trigger controller
func NewPipelineController(svc model.TriggerReaderWriter) *PipelineController {
	return &PipelineController{svc}
}

// ListPipelines get trigger pipelines
func (c *PipelineController) ListPipelines(ctx *gin.Context) {
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
