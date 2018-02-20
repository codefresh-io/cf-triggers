package model

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// PipelineRun pipeline run with ID (can be empty on error) and error (when failed)
	PipelineRun struct {
		ID    string `json:"id" yaml:"id"`
		Error error  `json:"error, omitempty" yaml:"error,omitempty"`
	}

	// Trigger a single link between event and pipeline
	Trigger struct {
		// unique event URI, use ':' instead of '/'
		Event string `json:"event" yaml:"event"`
		// pipeline
		Pipeline string `json:"pipeline" yaml:"pipeline"`
	}

	// TriggerReaderWriter interface
	TriggerReaderWriter interface {
		// trigger events
		GetEvent(event string) (*Event, error)
		GetEvents(eventType, kind, filter string) ([]Event, error)
		CreateEvent(eventType, kind, secret string, context string, values map[string]string) (*Event, error)
		DeleteEvent(event string, context string) error
		GetSecret(eventURI string) (string, error)
		// triggers
		ListTriggersForEvents(events []string) ([]Trigger, error)
		CreateTriggersForEvent(event string, pipelines []string) error
		DeleteTriggersForEvent(event string, pipelines []string) error
		// pipelines
		ListTriggersForPipelines(pipelines []string) ([]Trigger, error)
		GetPipelinesForTriggers(events []string) ([]string, error)
		CreateTriggersForPipeline(pipeline string, events []string) error
		DeleteTriggersForPipeline(pipeline string, events []string) error
	}

	// Runner pipeline runner
	Runner interface {
		Run(pipelines []string, vars map[string]string) ([]PipelineRun, error)
	}

	// Pinger ping response
	Pinger interface {
		Ping() (string, error)
	}

	// SecretChecker validates message secret or HMAC signature
	SecretChecker interface {
		Validate(message string, secret string, key string) error
	}
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
