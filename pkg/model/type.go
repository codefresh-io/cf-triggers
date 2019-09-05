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
		// Type field type (default 'string'): string, int, date, list, cron
		Type string `json:"type,omitempty" yaml:"type,omitempty"`
		// Help text
		Help string `json:"help,omitempty" yaml:"help,omitempty"`
		// Options an options map (key: value) for list type
		Options map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
		// Validator validator for value: regex, '|' separated list, int range, date range, http-get
		Validator string `json:"validator,omitempty" yaml:"validator,omitempty"`
		// Required required flag (default: false)
		Required bool `json:"required,omitempty" yaml:"required,omitempty"`
		// NameLabel name-label filed
		NameLabel string `json:"name-label" yaml:"name-label"`
	}

	// FilterField configuration field
	FilterField struct {
		// Name field name
		Name string `json:"name" yaml:"name"`
		// Type field type (default 'string'): string, int, date, list, crontab
		Type string `json:"type,omitempty" yaml:"type,omitempty"`
		// Help text
		Help string `json:"help,omitempty" yaml:"help,omitempty"`
		// Validator validator for value: regex, '|' separated list, int range, date range, http-get
		Validator string `json:"validator,omitempty" yaml:"validator,omitempty"`
	}

	// EventType event type
	EventType struct {
		// Event type name; e.g. registry, git
		Type string `json:"type" yaml:"type"`
		// Event Handler service url
		ServiceURL string `json:"service-url" yaml:"service-url"`
		//Event kind name; e.g. dockerhub|ecr|gcr (registry), github|bitbucket|gitlab (git)
		Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
		// URI template; e.g. registry:dockerhub:{{namespace}}:{{name}}:push
		URITemplate string `json:"uri-template,omitempty" yaml:"uri-template,omitempty"`
		// URI pattern; event uri match pattern - helps to detect type and kind from uri
		URIPattern string `json:"uri-regex" yaml:"uri-regex"`
		// Help URL
		HelpURL string `json:"help-url,omitempty" yaml:"help-url,omitempty"`
		// Configuration Fields
		Config []ConfigField `json:"config" yaml:"config"`
		// Filters - fields that support filtering
		Filters []FilterField `json:"filters" yaml:"filters"`
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
