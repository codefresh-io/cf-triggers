package model

import (
	"crypto/sha1"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"

	"github.com/codefresh-io/hermes/pkg/util"
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
		Kind string `json:"kind" yaml:"kind"`
		// event account
		Account string `json:"account" yaml:"account"`
		// event secret, used for event validation
		Secret string `json:"secret" yaml:"secret"`
	}
)

const (
	// OriginalPayload field name
	OriginalPayload = "EVENT_PAYLOAD"
)

// PublicAccount public account ID [0]{12}
var PublicAccount = strings.Repeat("0", 12)

// PublicAccountHash calculated sha1 hash
var PublicAccountHash = CalculateAccountHash(PublicAccount)

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
		URI:     strings.TrimPrefix(event, "event:"),
		Type:    fields["type"],
		Kind:    fields["kind"],
		Account: fields["account"],
		Secret:  fields["secret"],
		EventInfo: EventInfo{
			Endpoint:    fields["endpoint"],
			Description: fields["description"],
			Help:        fields["help"],
			Status:      fields["status"],
		},
	}
}

// CalculateAccountHash return first 12 of account SHA1 hash
// return empty string for empty account
func CalculateAccountHash(account string) string {
	// if account is already SHA1 hash
	// if it contains exactly 12 hex characters -> make an assumption about it being a hash
	if len(account) == 12 && util.IsHexString(account) {
		return account
	}
	// calculate SHA1 hash
	hex := fmt.Sprintf("%x", sha1.Sum([]byte(account)))
	runes := []rune(hex)
	// return first 12 characters only
	return string(runes[0:12])
}

// MatchAccount match account for passed uri
func MatchAccount(account, uri string) bool {
	hash := CalculateAccountHash(account)
	return strings.HasSuffix(uri, hash)
}

// MatchPublicAccount match public account for passed uri
func MatchPublicAccount(uri string) bool {
	return strings.HasSuffix(uri, PublicAccountHash)
}
