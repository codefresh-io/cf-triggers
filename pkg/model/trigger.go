package model

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
	}
)

// IsEmpty check if trigger is empty
func (m Trigger) IsEmpty() bool {
	return m.Event == ""
}
