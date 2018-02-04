package model

import (
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

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
		// Event Handler service url
		ServiceURL string `json:"service-url" yaml:"service-url"`
		//Event kind name; e.g. dockerhub|ecr|gcr (registry), github|bitbucket|gitlab (git)
		Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
		// URI template; e.g. index.docker.io:{{namespace}}:{{name}}:push
		URITemplate string `json:"uri-template,omitempty" yaml:"uri-template,omitempty"`
		// URI pattern; event uri match pattern - helps to detect type and kind from uri
		URIPattern string `json:"uri-regex" yaml:"uri-regex"`
		// Configuration Fields
		Config []ConfigField `json:"config,omitempty" yaml:"config,omitempty"`
	}

	// EventTypes array of event types
	EventTypes struct {
		Types []EventType `json:"types" yaml:"types"`
	}
)

// String retrun event type as YAML string
func (t EventType) String() string {
	d, err := yaml.Marshal(&t)
	if err != nil {
		log.WithError(err).Error("Failed to convert EventType to YAML")
	}
	return string(d)
}
