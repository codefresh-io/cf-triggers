package controller

import (
	"context"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
	"github.com/gin-gonic/gin"

	"github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrgin/v1"
	log "github.com/sirupsen/logrus"
)

// RunnerController trigger controller
type RunnerController struct {
	runnerSvc    model.Runner
	publisherSvc model.EventPublisher
	eventSvc     model.TriggerEventReaderWriter
	triggerSvc   model.TriggerReaderWriter
}

// NewRunnerController new runner controller
func NewRunnerController(runnerSvc model.Runner, publisherSvc model.EventPublisher, eventSvc model.TriggerEventReaderWriter, triggerSvc model.TriggerReaderWriter) *RunnerController {
	return &RunnerController{
		runnerSvc:    runnerSvc,
		publisherSvc: publisherSvc,
		eventSvc:     eventSvc,
		triggerSvc:   triggerSvc,
	}
}

// RunTrigger pipelines for trigger
func (c *RunnerController) RunTrigger(ctx *gin.Context) {
	// get trigger event
	event := getParam(ctx, "event")
	log.WithField("event", event).Debug("triggering pipelines for event")
	// get event payload
	var normEvent model.NormalizedEvent
	if err := ctx.BindJSON(&normEvent); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResult{http.StatusBadRequest, "error in JSON body", err.Error()})
		return
	}
	// get NewRelic transaction
	txn := nrgin.Transaction(ctx)
	// prepare context for specified event (skip account check)
	allCtx := context.WithValue(context.Background(), model.ContextKeyAccount, "-")
	// add NewRelic transaction to context if not nil
	if txn != nil {
		allCtx = context.WithValue(allCtx, model.ContextNewRelicTxn, txn)
	}
	// get trigger event
	triggerEvent, err := c.eventSvc.GetEvent(allCtx, event)
	if err != nil {
		status := http.StatusInternalServerError
		if err == model.ErrTriggerNotFound {
			status = http.StatusNotFound
		}
		ctx.JSON(status, ErrorResult{status, "failed to get event", err.Error()})
		return
	}
	// report event to eventbus
	err = c.publisherSvc.Publish(allCtx, triggerEvent.Account, event, normEvent)
	if err != nil {
		// on error report to log and continue
		log.WithError(err).Error("failed to publish event to eventbus")
	}
	// add original payload to variables
	vars := make(map[string]string)
	for k, v := range normEvent.Variables {
		vars[k] = v
	}
	vars["EVENT_PAYLOAD"] = normEvent.Original
	// get connected pipelines
	pipelines, err := c.triggerSvc.GetTriggerPipelines(allCtx, event, vars)
	if err != nil {
		// if there are no pipelines connected to the trigger event don't fail this REST method
		// to avoid multiple 'errors' reported to the event provider log
		// it's possible to have trigger event defined and not connected to any pipeline
		if err == model.ErrPipelineNotFound || err == model.ErrTriggerNotFound {
			log.WithField("event", event).Warn("there are no pipelines associated with trigger event")
			ctx.Status(http.StatusNoContent)
			return
		}
		ctx.JSON(http.StatusInternalServerError, ErrorResult{http.StatusInternalServerError, "failed to run trigger pipelines", err.Error()})
		return
	}
	// record execution history without run IDS
	log.WithFields(log.Fields{
		"account":   triggerEvent.Account,
		"event":     triggerEvent.URI,
		"pipelines": pipelines,
	}).Info("going to run pipelines for trigger event")
	// record NewRelic segment for Run
	if txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// run piplines
	runs, err := c.runnerSvc.Run(triggerEvent.Account, pipelines, vars, normEvent)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResult{http.StatusInternalServerError, "failed to run trigger pipelines", err.Error()})
		return
	}
	// record execution history with run IDS
	log.WithFields(log.Fields{
		"account":   triggerEvent.Account,
		"event":     triggerEvent.URI,
		"pipelines": pipelines,
		"runs":      runs,
	}).Info("pipelines for trigger event are running")
	// return ok status with run IDS
	ctx.JSON(http.StatusOK, runs)
}
