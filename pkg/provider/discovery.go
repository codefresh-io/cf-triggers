package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"

	log "github.com/sirupsen/logrus"
	"path/filepath"
)

type (
	// EventProviderManager is responsible for discovering new Trigger Event Providers
	EventProviderManager struct {
		sync.RWMutex
		configFile string
		eventTypes model.EventTypes
		watcher    *util.FileWatcher
	}

	// EventProvider describes installed and configured Trigger Event Providers
	EventProvider interface {
		GetTypes() []model.EventType
		MatchType(eventURI string) (*model.EventType, error)
		GetType(t string, k string) (*model.EventType, error)
		GetEventInfo(eventURI string, secret string) (*model.EventInfo, error)
		SubscribeToEvent(event, secret string, credentials map[string]string) (*model.EventInfo, error)
		UnsubscribeFromEvent(event string, credentials map[string]string) error
		ConstructEventURI(t string, k string, values map[string]string) (string, error)
	}
)

var (
	instance *EventProviderManager
	once     sync.Once
)

// non singleton - for test only
func newTestEventProviderManager(configFile string) *EventProviderManager {
	instance = new(EventProviderManager)
	instance.configFile = configFile
	// start monitoring
	instance.eventTypes, _ = loadEventHandlerTypes(configFile)
	instance.watcher = instance.monitorConfigFile()
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
		log.WithError(err).Error("Failed to read config file (provided path is illegal)")
		return eventTypes, err
	}

	eventTypesData, err := ioutil.ReadFile(absConfigFilePath)
	if err != nil {
		log.WithError(err).Error("Failed to read config file")
		return eventTypes, err
	}

	err = json.Unmarshal(eventTypesData, &eventTypes)
	if err != nil {
		log.WithError(err).Error("Failed to load types configuration from JSON file")
		return eventTypes, err
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
			log.WithError(err).Error("Failed to load config file")
		}
	})
	if err != nil {
		log.WithError(err).Error("Failed to watch file for changes")
	}

	return watcher
}

// GetTypes get discovered event provider types
func (m *EventProviderManager) GetTypes() []model.EventType {
	m.Lock()
	defer m.Unlock()
	if len(m.eventTypes.Types) != 0 {
		return m.eventTypes.Types
	}

	log.Error("Failed to fetch event types")
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
	return nil, errors.New("failed to find event type")
}

// MatchType match event type by uri
func (m *EventProviderManager) MatchType(eventURI string) (*model.EventType, error) {
	log.WithField("event-uri", eventURI).Debug("Matching event type")

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
		if r.MatchString(eventURI) {
			return &e, nil
		}
	}

	return nil, errors.New("failed to match event type")
}

// GetEventInfo get event info from event provider
func (m *EventProviderManager) GetEventInfo(event string, secret string) (*model.EventInfo, error) {
	log.WithField("event-uri", event).Debug("Getting event info from event provider")
	et, err := m.MatchType(event)
	if err != nil {
		return nil, err
	}

	// call Event Provider service to get event info
	provider := NewEventProviderEndpoint(et.ServiceURL)
	info, err := provider.GetEventInfo(event, secret)
	if err != nil {
		log.WithError(err).Error("Failed to get event info")
		return nil, err
	}

	return info, nil
}

// SubscribeToEvent subscribe to remote event through event provider
func (m *EventProviderManager) SubscribeToEvent(event, secret string, credentials map[string]string) (*model.EventInfo, error) {
	log.WithField("event-uri", event).Debug("Subscribe to remote event trough event provider")
	et, err := m.MatchType(event)
	if err != nil {
		return nil, err
	}

	// call Event Provider service to subscribe to remote event
	provider := NewEventProviderEndpoint(et.ServiceURL)
	info, err := provider.SubscribeToEvent(event, secret, credentials)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// UnsubscribeFromEvent unsubscribe from remote event through event provider
func (m *EventProviderManager) UnsubscribeFromEvent(event string, credentials map[string]string) error {
	log.WithField("event-uri", event).Debug("Unsubscribe from remote event trough event provider")
	et, err := m.MatchType(event)
	if err != nil {
		return err
	}

	// call Event Provider service to subscribe to remote event
	provider := NewEventProviderEndpoint(et.ServiceURL)
	return provider.UnsubscribeFromEvent(event, credentials)
}

// ConstructEventURI construct event URI from type/kind and values map
func (m *EventProviderManager) ConstructEventURI(t string, k string, values map[string]string) (string, error) {
	log.WithFields(log.Fields{
		"type":   t,
		"kind":   k,
		"values": values,
	}).Debug("constructing event URI")

	// get event type
	eventType, err := m.GetType(t, k)
	if err != nil {
		return "", err
	}

	// event URI is set to URI template initially
	eventURI := eventType.URITemplate

	// scan through all config fields
	for _, field := range eventType.Config {
		// get value for config field name
		val := values[field.Name]
		// validate value
		log.WithFields(log.Fields{
			"field": field.Name,
			"regex": field.Validator,
		}).Debug("validating field")
		r, err := regexp.Compile(field.Validator)
		if err != nil {
			return "", err
		}
		if !r.MatchString(val) {
			return "", errors.New("field validation failed")
		}
		// substitute value for template string in URI template
		eventURI = strings.Replace(eventURI, fmt.Sprintf("{{%s}}", field.Name), val, -1)
	}

	// do a final validation
	r, err := regexp.Compile(eventType.URIPattern)
	if err != nil {
		return "", err
	}
	if r.MatchString(eventURI) {
		return eventURI, nil
	}
	return "", errors.New("event URI does not match URI pattern")
}
