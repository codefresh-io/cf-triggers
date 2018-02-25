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
	event := getParam(ctx, "event")
	var pipelines []string
	var err error
	if pipelines, err = c.svc.GetPipelinesForTriggers([]string{event}, ctx.Query("account")); err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to get list pipelines", err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, pipelines)
}
