package model

import (
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type (
	// EventInfo - single trigger event
	EventInfo struct {
		// URI event unique identifier
		URI string `json:"uri" yaml:"uri"`
		// event secret, used for event validation
		Secret string `json:"secret" yaml:"secret"`
		// Endpoint URL
		Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
		// Description human readable text
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
		// Status current event handler status (active, error, not active)
		Status string `json:"status,omitempty" yaml:"status,omitempty"`
		// Help text
		Help string `json:"help,omitempty" yaml:"help,omitempty"`
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
