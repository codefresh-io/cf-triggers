package model

import "reflect"

type (
	// Pipeline Codefresh Pipeline URI
	Pipeline struct {
		// pipeline name
		Name string `json:"name"`
		// Git repository owner
		RepoOwner string `json:"repo-owner"`
		// Git repository name
		RepoName string `json:"repo-name"`
	}

	// Trigger describes a trigger type
	Trigger struct {
		// unique event URI, using ':' instead of '/'
		Event string `json:"event"`
		// trigger secret
		Secret string `json:"secret"`
		// pipelines
		Pipelines []Pipeline `json:"pipelines"`
	}

	// TriggerService interface
	TriggerService interface {
		List(filter string) ([]Trigger, error)
		Get(id string) (Trigger, error)
		Add(Trigger) error
		Delete(id string) error
		Update(Trigger) error
		Run(id string, vars map[string]string) error
		CheckSecret(id string, secret string) error
	}
)

// EmptyTrigger is empty trigger to reuse
var EmptyTrigger = Trigger{}

// IsEmpty check if trigger is empty
func (m Trigger) IsEmpty() bool {
	return reflect.DeepEqual(m, EmptyTrigger)
}
