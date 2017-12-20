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
		// Git repository owner
		RepoOwner string `json:"repo-owner" yaml:"repo-owner"`
		// Git repository name
		RepoName string `json:"repo-name" yaml:"repo-name"`
		// pipeline name
		Name string `json:"name" yaml:"name"`
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

	// TriggerService interface
	TriggerService interface {
		List(filter string) ([]*Trigger, error)
		Get(id string) (*Trigger, error)
		Add(trigger Trigger) error
		Delete(id string) error
		Update(trigger Trigger) error
		GetPipelines(id string) ([]Pipeline, error)
		AddPipelines(id string, pipelines []Pipeline) error
		DeletePipeline(id string, pid string) error
		Run(id string, vars map[string]string) ([]string, error)
		CheckSecret(id string, message string, secret string) error
		Ping() (string, error)
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
		log.Errorf("error: %v", err)
	}
	return string(d)
}

// String retrun trigger as YAML string
func (p Pipeline) String() string {
	d, err := yaml.Marshal(&p)
	if err != nil {
		log.Errorf("error: %v", err)
	}
	return string(d)
}

// PipelineFromURI construct pipeline struct from uri - repo-owner:repo-name:name
func PipelineFromURI(uri string) (*Pipeline, error) {
	s := strings.Split(uri, ":")
	if len(s) != 3 { // should be name:repo-owner:repo-name
		return nil, errors.New("invalid pipeline uri, should be in form: [repo-owner]:[repo-name]:[name]")
	}
	return &Pipeline{RepoOwner: s[0], RepoName: s[1], Name: s[2]}, nil
}

// PipelineToURI convert pipeline struct to pipeline-uri
func PipelineToURI(p *Pipeline) string {
	return strings.Join([]string{p.RepoOwner, p.RepoName, p.Name}, ":")
}
