package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/sling"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"

	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type (
	// EventProviderManager is responsible for discovering new Trigger Event Providers
	EventProviderManager struct {
		sync.RWMutex
		configFile string
		eventTypes model.EventTypes
		watcher    *util.FileWatcher
		// test related fields
		testMode bool
		testDoer sling.Doer
	}

	// EventProvider describes installed and configured Trigger Event Providers
	EventProvider interface {
		GetTypes() []model.EventType
		MatchType(eventURI string) (*model.EventType, error)
		GetType(t string, k string) (*model.EventType, error)
		GetEventInfo(ctx context.Context, eventURI string, secret string) (*model.EventInfo, error)
		SubscribeToEvent(ctx context.Context, eventURI, secret string, actions []string, credentials map[string]interface{}) (*model.EventInfo, error)
		UnsubscribeFromEvent(ctx context.Context, eventURI string, credentials map[string]interface{}) error
		ConstructEventURI(t string, k string, a string, values map[string]string) (string, error)
	}
)

var (
	instance *EventProviderManager
	once     sync.Once
)

// non singleton - for test only
func newTestEventProviderManager(configFile string, doer sling.Doer) *EventProviderManager {
	instance = new(EventProviderManager)
	instance.configFile = configFile
	// start monitoring
	instance.eventTypes, _ = loadEventHandlerTypes(configFile)
	instance.watcher = instance.monitorConfigFile()
	// set test mode
	instance.testMode = true
	instance.testDoer = doer
	// return it
	return instance
}

// NewEventProviderManager return new Event Handler Manager (singleton)
// Event Handler Manager discoveres all registered Event Handlers and can describe them
func NewEventProviderManager(configFile string, skipMonitor bool) *EventProviderManager {
	once.Do(func() {
		instance = new(EventProviderManager)
		instance.configFile = configFile
		// load config file
		log.WithFields(log.Fields{
			"config":       configFile,
			"skip-monitor": skipMonitor,
		}).Debug("Loading types configuration (first time)")
		var err error
		instance.eventTypes, err = loadEventHandlerTypes(configFile)
		if err != nil {
			log.WithError(err).Error("Failed to load types configuration")
		}
		// start monitoring
		if !skipMonitor {
			log.Debug("Starting monitor config types file for updates")
			instance.watcher = instance.monitorConfigFile()
		}
	})
	// return it
	return instance
}

// load EventHandler Types
func loadEventHandlerTypes(configFile string) (model.EventTypes, error) {
	eventTypes := model.EventTypes{}

	absConfigFilePath, err := filepath.Abs(configFile)
	if err != nil {
		log.WithError(err).Error("failed to read config file (provided path is illegal)")
		return eventTypes, err
	}

	eventTypesData, err := ioutil.ReadFile(absConfigFilePath)
	if err != nil {
		log.WithError(err).Error("failed to read config file")
		return eventTypes, err
	}

	err = json.Unmarshal(eventTypesData, &eventTypes)
	if err != nil {
		log.WithError(err).Error("Failed to load types configuration from JSON file")
		return eventTypes, err
	}
	// scan through all types and add optional account short hash
	for i, et := range eventTypes.Types {
		// trim last '$' character if exists
		et.URIPattern = strings.TrimSuffix(et.URIPattern, "$")
		// add optional 12 hexadecimal string (short account SHA1 hash code)
		eventTypes.Types[i].URIPattern = et.URIPattern + "(:[[:xdigit:]]{12})$"
	}
	return eventTypes, nil
}

// Close - free file watcher resources
func (m *EventProviderManager) Close() {
	if m.watcher != nil {
		log.Debug("Close file watcher")
		m.watcher.Close()
	}
}

// NOTE: should be called only once
// monitor configuration directory to discover new/updated/deleted Event Handlers
func (m *EventProviderManager) monitorConfigFile() *util.FileWatcher {
	// Watch the file for modification and update the config manager with the new config when it's available
	watcher, err := util.WatchFile(m.configFile, time.Second, func() {
		log.Debug("Config types file updated")
		m.Lock()
		defer m.Unlock()
		var err error
		m.eventTypes, err = loadEventHandlerTypes(m.configFile)
		if err != nil {
			log.WithError(err).Error("failed to load config file")
		}
	})
	if err != nil {
		log.WithError(err).Error("failed to watch file for changes")
	}

	return watcher
}

// read values map from uri
func (m *EventProviderManager) getValuesFromURI(uri string) (map[string]string, error) {
	log.WithField("event", uri).Debug("getting values map from event URI")
	et, err := m.MatchType(uri)
	if err != nil {
		log.WithError(err).Error("failed to find event type")
		return nil, err
	}
	// split URI
	parts := strings.Split(uri, ":")
	// drop last part - account hash
	if len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	// split URI template
	tokens := strings.Split(et.URITemplate, ":")
	// event URI (w/out account) must match template (have same number of tokens)
	if len(tokens) != len(parts) {
		log.WithError(err).Error("event URI does not match URI template")
		return nil, err
	}
	// scan through tokens and create map for every '{{token}}' (template token)
	values := make(map[string]string)
	for index, token := range tokens {
		// if template token -> store it in values map
		if strings.HasPrefix(token, "{{") && strings.HasSuffix(token, "}}") {
			token = strings.TrimPrefix(token, "{{")
			token = strings.TrimSuffix(token, "}}")
			values[token] = parts[index]
		}
	}
	return values, nil
}

// GetTypes get discovered event provider types
func (m *EventProviderManager) GetTypes() []model.EventType {
	m.Lock()
	defer m.Unlock()
	if len(m.eventTypes.Types) != 0 {
		return m.eventTypes.Types
	}

	log.Error("failed to fetch event types")
	return nil
}

// GetType get individual event provider type (by type and kind)
func (m *EventProviderManager) GetType(eventType string, eventKind string) (*model.EventType, error) {
	log.WithFields(log.Fields{
		"type": eventType,
		"kind": eventKind,
	}).Debug("get event type")

	m.Lock()
	defer m.Unlock()

	for _, e := range m.eventTypes.Types {
		if e.Type == eventType && e.Kind == eventKind {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("failed to find event type '%s' kind '%s'", eventType, eventKind)
}

// MatchType match event type by uri
func (m *EventProviderManager) MatchType(event string) (*model.EventType, error) {
	log.WithField("event", event).Debug("matching event type")

	m.Lock()
	defer m.Unlock()

	for _, e := range m.eventTypes.Types {
		r, err := regexp.Compile(e.URIPattern)
		if err != nil {
			log.WithFields(log.Fields{
				"type":  e.Type,
				"regex": e.URIPattern,
			}).Error("bad uri regex pattern for type")
			continue // skip
		}
		if r.MatchString(event) {
			return &e, nil
		}
	}

	return nil, errors.New("failed to match event type")
}

// GetEventInfo get event info from event provider
func (m *EventProviderManager) GetEventInfo(ctx context.Context, event string, secret string) (*model.EventInfo, error) {
	log.WithField("event", event).Debug("getting event info from event provider")
	et, err := m.MatchType(event)
	if err != nil {
		return nil, err
	}

	// call Event Provider service to get event info
	var provider EventProviderService
	if m.testMode {
		provider = newTestEventProviderEndpoint(m.testDoer, et.ServiceURL)
	} else {
		provider = NewEventProviderEndpoint(et.ServiceURL)
	}
	info, err := provider.GetEventInfo(ctx, event, secret)
	if err != nil {
		log.WithError(err).Error("failed to get event info")
		return nil, err
	}

	return info, nil
}

// SubscribeToEvent subscribe to remote event through event provider
func (m *EventProviderManager) SubscribeToEvent(ctx context.Context, eventURI, secret string, actions []string, credentials map[string]interface{}) (*model.EventInfo, error) {
	log.WithField("event", eventURI).Debug("subscribe to remote event trough event provider")
	et, err := m.MatchType(eventURI)
	if err != nil {
		log.WithError(err).Error("failed to find event type")
		return nil, err
	}
	// get values from uri
	values, err := m.getValuesFromURI(eventURI)
	if err != nil {
		log.WithError(err).Error("failed to get values from uri")
		return nil, err
	}

	// call Event Provider service to subscribe to remote event
	var provider EventProviderService
	if m.testMode {
		provider = newTestEventProviderEndpoint(m.testDoer, et.ServiceURL)
	} else {
		provider = NewEventProviderEndpoint(et.ServiceURL)
	}
	info, err := provider.SubscribeToEvent(ctx, eventURI, et.Type, et.Kind, secret, actions, values, credentials)
	if err != nil {
		log.WithError(err).Error("failed to subscribe to event")
		return nil, err
	}

	return info, nil
}

// UnsubscribeFromEvent unsubscribe from remote event through event provider
func (m *EventProviderManager) UnsubscribeFromEvent(ctx context.Context, eventURI string, credentials map[string]interface{}) error {
	log.WithField("event", eventURI).Debug("unsubscribe from remote event trough event provider")
	et, err := m.MatchType(eventURI)
	if err != nil {
		log.WithError(err).Error("failed to match event type")
		return err
	}
	// get values from uri
	values, err := m.getValuesFromURI(eventURI)
	if err != nil {
		log.WithError(err).Error("failed to get values from uri")
		return err
	}

	// call Event Provider service to subscribe to remote event
	var provider EventProviderService
	if m.testMode {
		provider = newTestEventProviderEndpoint(m.testDoer, et.ServiceURL)
	} else {
		provider = NewEventProviderEndpoint(et.ServiceURL)
	}
	err = provider.UnsubscribeFromEvent(ctx, eventURI, et.Type, et.Kind, values, credentials)
	if err != nil {
		log.WithError(err).Error("failed to unsubscribe from the event")
	}
	return nil
}

// ConstructEventURI construct event URI from type/kind, account and values map
func (m *EventProviderManager) ConstructEventURI(t string, k string, a string, values map[string]string) (string, error) {
	log.WithFields(log.Fields{
		"type":   t,
		"kind":   k,
		"values": values,
	}).Debug("constructing event URI")

	// get event type
	eventType, err := m.GetType(t, k)
	if err != nil {
		log.WithError(err).Error("failed to find trigger type")
		return "", err
	}

	// event URI is set to URI template initially
	event := eventType.URITemplate

	// scan through all config fields
	for _, field := range eventType.Config {
		// get value for config field name
		val := values[field.Name]
		// validate value
		log.WithFields(log.Fields{
			"field": field.Name,
			"regex": field.Validator,
		}).Debug("validating field")
		r, e := regexp.Compile(field.Validator)
		if e != nil {
			log.WithError(e).WithField("regex", field.Validator).Error("failed to compile validator regex")
			return "", e
		}
		if !r.MatchString(val) {
			log.WithField("field", field.Name).Error("field validation failed")
			return "", fmt.Errorf("field '%s' validation failed for validator '%s'", field.Name, field.Validator)
		}
		// substitute value for template string in URI template
		event = strings.Replace(event, fmt.Sprintf("{{%s}}", field.Name), val, -1)
	}
	// append account short (12 hex chars) SHA1 if non-empty
	hash := model.CalculateAccountHash(a)
	event = fmt.Sprintf("%s:%s", event, hash)

	// do a final validation
	r, err := regexp.Compile(eventType.URIPattern)
	if err != nil {
		log.WithError(err).WithField("regex", eventType.URIPattern).Error("failed to compile URI regex")
		return "", err
	}
	if r.MatchString(event) {
		return event, nil
	}
	log.Error("event URI does not match URI pattern")
	return "", fmt.Errorf("event '%s' does not match trigger type URI pattern", event)
}
