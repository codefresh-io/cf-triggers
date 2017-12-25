package model

type (
	// ConfigField configuration field
	ConfigField struct {
		// Name field name
		Name string `json:"name" yaml:"name"`
		// Type field type (default 'string'): string, int, date, list, crontab
		Type string `json:"type,omitempty" yaml:"type,omitempty"`
		// Validator validator for value: regex, '|' separated list, int range, date range, http-get
		Validator string `json:"validator,omitempty" yaml:"validator,omitempty"`
		// Required required flag (default: false)
		Required bool `json:"required,omitempty" yaml:"required,omitempty"`
	}

	// EventType event type
	EventType struct {
		// Event type name; e.g. registry, git
		Type string `json:"type" yaml:"type"`
		//Event kind name; e.g. dockerhub/ecr/gcr (registry), github/bitbucket/gitlab (git)
		Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
		// URI template; e.g. index.docker.io:{{namespace}}:{{name}}:push
		URITemplate string `json:"uri-template" yaml:"uri-template"`
		// Configuration Fields
		Config []ConfigField `json:"config" yaml:"config"`
	}

	// EventInfo event info - EVERYTHING for specific event (eventURI)
	EventInfo struct {
		// Endpoint URL
		Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
		// Description human readable text
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
		// Status current event handler status (active, error, not active)
		Status string `json:"status,omitempty" yaml:"status,omitempty"`
	}
)
