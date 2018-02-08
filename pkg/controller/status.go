package controller

import (
	"net/http"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/version"
	"github.com/gin-gonic/gin"
)

// StatusController status controller
type StatusController struct {
	backend   model.Pinger
	codefresh codefresh.PipelineService
}

// NewStatusController init status controller
func NewStatusController(backend model.Pinger, codefresh codefresh.PipelineService) *StatusController {
	return &StatusController{backend, codefresh}
}

// GetHealth status
func (c *StatusController) GetHealth(ctx *gin.Context) {
	// check backend store
	_, err := c.backend.Ping()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to talk to backend", "error": err.Error()})
		return
	}
	// check codefresh service
	err = c.codefresh.Ping()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "failed to talk to Codefresh API", "error": err.Error()})
		return
	}
	// everything is good
	ctx.String(http.StatusOK, "Healthy")
}

// Ping return PONG with OK
func (c *StatusController) Ping(ctx *gin.Context) {
	ctx.String(http.StatusOK, "PONG")
}

// GetVersion get app version
func (c *StatusController) GetVersion(ctx *gin.Context) {
	ctx.String(http.StatusOK, version.WebVersion)
}
