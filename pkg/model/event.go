package model

import (
	"strings"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type (
	// EventInfo event info as seen by trigger provider
	EventInfo struct {
		// Endpoint URL
		Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
		// Description human readable text
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
		// Status of current event for event provider (active, error, not active)
		Status string `json:"status,omitempty" yaml:"status,omitempty"`
		// Help text
		Help string `json:"help,omitempty" yaml:"help,omitempty"`
	}

	// Event single trigger event
	Event struct {
		// event info
		EventInfo
		// URI event unique identifier
		URI string `json:"uri" yaml:"uri"`
		// event type
		Type string `json:"type" yaml:"type"`
		// event kind
		Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
		// event secret, used for event validation
		Secret string `json:"secret" yaml:"secret"`
	}
)

// String retrun event info as YAML string
func (t EventInfo) String() string {
	d, err := yaml.Marshal(&t)
	if err != nil {
		log.WithError(err).Error("Failed to convert EventInfo to YAML")
	}
	return string(d)
}

// String retrun event info as YAML string
func (t Event) String() string {
	d, err := yaml.Marshal(&t)
	if err != nil {
		log.WithError(err).Error("Failed to convert Event to YAML")
	}
	return string(d)
}

// StringsMapToEvent convert map[string]string to Event
func StringsMapToEvent(event string, fields map[string]string) *Event {
	return &Event{
		URI:    strings.TrimPrefix(event, "event:"),
		Type:   fields["type"],
		Kind:   fields["kind"],
		Secret: fields["secret"],
		EventInfo: EventInfo{
			Endpoint:    fields["endpoint"],
			Description: fields["description"],
			Help:        fields["help"],
			Status:      fields["status"],
		},
	}
}
