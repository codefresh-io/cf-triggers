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

	// TriggerReaderWriter interface
	TriggerReaderWriter interface {
		List(filter string) ([]*Trigger, error)
		ListByPipeline(pipelineUID string) ([]*Trigger, error)
		GetSecret(eventURI string) (string, error)
		Get(eventURI string) (*Trigger, error)
		Add(trigger Trigger) error
		Delete(eventURI string) error
		Update(trigger Trigger) error
		GetPipelines(eventURI string) ([]string, error)
		AddPipelines(eventURI string, pipelines []string) error
		DeletePipeline(eventURI string, pipelineUID string) error
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
