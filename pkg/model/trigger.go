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

	// Trigger describes a trigger type
	Trigger struct {
		// unique event URI, use ':' instead of '/'
		Event string `json:"event" yaml:"event"`
		// trigger secret
		Secret string `json:"secret" yaml:"secret"`
		// pipelines
		Pipelines []string `json:"pipelines" yaml:"pipelines"`
	}

	TriggerLink struct {
		// unique event URI, use ':' instead of '/'
		Event string `json:"event" yaml:"event"`
		// pipeline
		Pipeline string `json:"pipeline" yaml:"pipeline"`
	}

	// TriggerReaderWriter interface
	TriggerReaderWriter interface {
		// trigger events
		GetEvent(event string) (*Event, error)
		CreateEvent(eventType string, kind string, secret string, values map[string]string) (*Event, error)
		DeleteEvent(event string) error
		GetSecret(eventURI string) (string, error)
		// triggers
		ListTriggersForEvents(events []string) ([]TriggerLink, error)
		ListTriggersForPipelines(pipelines []string) ([]TriggerLink, error)
		GetPipelinesForTriggers(events []string) ([]string, error)
		CreateTriggersForEvent(event string, pipelines []string) error
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
