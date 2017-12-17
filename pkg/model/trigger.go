package model

import (
	"reflect"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// Pipeline Codefresh Pipeline URI
	Pipeline struct {
		// pipeline name
		Name string `json:"name" yaml:"name"`
		// Git repository owner
		RepoOwner string `json:"repo-owner" yaml:"repo-owner"`
		// Git repository name
		RepoName string `json:"repo-name" yaml:"repo-name"`
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
		List(filter string) ([]Trigger, error)
		Get(id string) (Trigger, error)
		Add(trigger Trigger) error
		Delete(id string) error
		Update(trigger Trigger) error
		Run(id string, vars map[string]string) ([]string, error)
		CheckSecret(id string, message string, secret string) error
		Ping() (string, error)
	}
)

// EmptyTrigger is empty trigger to reuse
var EmptyTrigger = Trigger{}

// GenerateKeyword keyword used to auto-generate secret
const GenerateKeyword = "!generate"

// IsEmpty check if trigger is empty
func (t Trigger) IsEmpty() bool {
	return reflect.DeepEqual(t, EmptyTrigger)
}

// String retrun trigger as YAML string
func (t Trigger) String() string {
	d, err := yaml.Marshal(&t)
	if err != nil {
		log.Errorf("error: %v", err)
	}
	return string(d)
}
