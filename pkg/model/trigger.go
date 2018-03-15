package model

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// PipelineRun pipeline run with ID (can be empty on error) and error (when failed)
	PipelineRun struct {
		ID    string `json:"id" yaml:"id"`
		Error error  `json:"error,omitempty" yaml:"error,omitempty"`
	}

	// Trigger a single link between event and pipeline
	Trigger struct {
		// unique event URI, use ':' instead of '/'
		Event string `json:"event" yaml:"event"`
		// pipeline
		Pipeline string `json:"pipeline" yaml:"pipeline"`
	}

	// TriggerEventReaderWriter interface
	TriggerEventReaderWriter interface {
		// trigger events
		GetEvent(ctx context.Context, event string) (*Event, error)
		GetEvents(ctx context.Context, eventType, kind, filter string) ([]Event, error)
		CreateEvent(ctx context.Context, eventType, kind, secret, context string, values map[string]string) (*Event, error)
		DeleteEvent(ctx context.Context, event, context string) error
	}

	// TriggerReaderWriter interface
	TriggerReaderWriter interface {
		// triggers
		GetEventTriggers(ctx context.Context, event string) ([]Trigger, error)
		GetPipelineTriggers(ctx context.Context, pipeline string) ([]Trigger, error)
		DeleteTrigger(ctx context.Context, event, pipeline string) error
		CreateTrigger(ctx context.Context, event, pipeline string) error
		GetTriggerPipelines(ctx context.Context, event string) ([]string, error)
	}

	// Runner pipeline runner
	Runner interface {
		Run(account string, pipelines []string, vars map[string]string) ([]PipelineRun, error)
	}

	// Pinger ping response
	Pinger interface {
		Ping() (string, error)
	}

	// SecretChecker validates message secret or HMAC signature
	SecretChecker interface {
		Validate(message string, secret string, key string) error
	}

	// contextKey type
	contextKey string
)

// Context keys
var (
	ContextKeyAccount = contextKey("account")
	ContextKeyUser    = contextKey("user")
	ContextKeyPublic  = contextKey("public")
	ContextRequestID  = contextKey("requestID")
	ContextAuthEntity = contextKey("authEntity")
)

// ErrTriggerNotFound error when trigger not found
var ErrTriggerNotFound = errors.New("trigger not found")

// ErrEventDeleteWithTriggers trigger-event has linked pipelines (omn delete)
var ErrEventDeleteWithTriggers = errors.New("cannot delete trigger event linked to pipelines")

// ErrEventNotFound error when trigger event not found
var ErrEventNotFound = errors.New("trigger event not found")

// ErrPipelineNotFound error when trigger not found
var ErrPipelineNotFound = errors.New("pipeline not found")

// ErrTriggerAlreadyExists error when trigger already exists
var ErrTriggerAlreadyExists = errors.New("trigger already exists")

// GenerateKeyword keyword used to auto-generate secret
const GenerateKeyword = "!generate"

// String retrun trigger as YAML string
func (t Trigger) String() string {
	d, err := yaml.Marshal(&t)
	if err != nil {
		log.WithError(err).Error("Failed to convert Trigger to YAML")
	}
	return string(d)
}
