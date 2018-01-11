package model

import (
	"errors"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// Pipeline Codefresh Pipeline URI
	Pipeline struct {
		// Codefresh account name
		Account string `json:"account" yaml:"account"`
		// Git repository owner
		RepoOwner string `json:"repo-owner" yaml:"repo-owner"`
		// Git repository name
		RepoName string `json:"repo-name" yaml:"repo-name"`
		// pipeline name
		Name string `json:"name" yaml:"name"`
	}

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
		Pipelines []Pipeline `json:"pipelines" yaml:"pipelines"`
	}

	// TriggerReaderWriter interface
	TriggerReaderWriter interface {
		List(filter string) ([]*Trigger, error)
		ListByPipeline(pipelineURI string) ([]*Trigger, error)
		GetSecret(eventURI string) (string, error)
		Get(eventURI string) (*Trigger, error)
		Add(trigger Trigger) error
		Delete(eventURI string) error
		Update(trigger Trigger) error
		GetPipelines(eventURI string) ([]Pipeline, error)
		AddPipelines(eventURI string, pipelines []Pipeline) error
		DeletePipeline(eventURI string, pipelineURI string) error
	}

	// Runner pipeline runner
	Runner interface {
		Run(pipelines []Pipeline, vars map[string]string) ([]PipelineRun, error)
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

// String retrun trigger as YAML string
func (p Pipeline) String() string {
	d, err := yaml.Marshal(&p)
	if err != nil {
		log.WithError(err).Error("Failed to convert Pipeline to YAML")
	}
	return string(d)
}

// PipelineFromURI construct pipeline struct from uri - repo-owner:repo-name:name
func PipelineFromURI(uri string) (*Pipeline, error) {
	s := strings.Split(uri, ":")
	if len(s) != 4 { // should be name:repo-owner:repo-name
		return nil, errors.New("invalid pipeline uri, should be in form: [account]:[repo-owner]:[repo-name]:[name]")
	}
	return &Pipeline{Account: s[0], RepoOwner: s[1], RepoName: s[2], Name: s[3]}, nil
}

// PipelineToURI convert pipeline struct to pipeline-uri
func PipelineToURI(p *Pipeline) string {
	return strings.Join([]string{p.Account, p.RepoOwner, p.RepoName, p.Name}, ":")
}
