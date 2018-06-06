package backend

import (
	"context"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	log "github.com/sirupsen/logrus"
)

// Publisher publish event to Codefresh eventbus
type Publisher struct {
	codefreshSvc codefresh.PipelineService
}

// NewPublisher initialize new Publisher
func NewPublisher(cf codefresh.PipelineService) model.EventPublisher {
	return &Publisher{cf}
}

// Publish Codefresh pipelines: return arrays of runs and errors
func (p *Publisher) Publish(ctx context.Context, account string, eventURI string, event model.NormalizedEvent) error {
	log.Debug("publishing event")
	return p.codefreshSvc.PublishEvent(ctx, account, eventURI, event)
}
