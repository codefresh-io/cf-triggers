package backend

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"sync"
	"time"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"

	log "github.com/sirupsen/logrus"
)

type (
	// EventHandlerManager is responsible for discovering new Event Handlers
	EventHandlerManager struct {
		sync.RWMutex
		configFile string
		eventTypes model.EventTypes
		watcher    *util.FileWatcher
	}

	// EventHandlerInformer describes installed and configured Event Handlers
	EventHandlerInformer interface {
		GetTypes() []model.EventType
		MatchType(eventURI string) *model.EventType
		GetType(t string, k string) *model.EventType
		GetEventInfo(eventURI string, secret string) *model.EventInfo
	}
)

var (
	instance *EventHandlerManager
	once     sync.Once
)

// non singleton - for test only
func newTestEventHandlerManager(configFile string) *EventHandlerManager {
	instance = new(EventHandlerManager)
	instance.configFile = configFile
	// start monitoring
	instance.eventTypes, _ = loadEventHandlerTypes(configFile)
	instance.watcher = instance.monitorConfigFile()
	// return it
	return instance
}

// NewEventHandlerManager return new Event Handler Manager (singleton)
// Event Handler Manager discoveres all registered Event Handlers and can describe them
func NewEventHandlerManager(configFile string, skipMonitor bool) *EventHandlerManager {
	once.Do(func() {
		instance = new(EventHandlerManager)
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
	eventTypesData, err := ioutil.ReadFile(configFile)
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
func (m *EventHandlerManager) Close() {
	if m.watcher != nil {
		log.Debug("Close file watcher")
		m.watcher.Close()
	}
}

// NOTE: should be called only once
// monitor configuration directory to discover new/updated/deleted Event Handlers
func (m *EventHandlerManager) monitorConfigFile() *util.FileWatcher {
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

// GetTypes get discovered event handler types
func (m *EventHandlerManager) GetTypes() []model.EventType {
	m.Lock()
	defer m.Unlock()
	if len(m.eventTypes.Types) != 0 {
		return m.eventTypes.Types
	}

	log.Error("Failed to fetch event types")
	return nil
}

// GetType get individual event handler type (by type and kind)
func (m *EventHandlerManager) GetType(eventType string, eventKind string) *model.EventType {
	m.Lock()
	defer m.Unlock()

	for _, e := range m.eventTypes.Types {
		if e.Type == eventType && e.Kind == eventKind {
			return &e
		}
	}

	log.WithFields(log.Fields{
		"type": eventType,
		"kind": eventKind,
	}).Error("fail to find event type")
	return nil
}

// MatchType match event type by uri
func (m *EventHandlerManager) MatchType(eventURI string) *model.EventType {
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
			return &e
		}
	}

	log.WithField("event-uri", eventURI).Errorf("Failed to find event type")
	return nil
}

// GetEventInfo get individual event handler type (by type and kind)
func (m *EventHandlerManager) GetEventInfo(eventURI string, secret string) *model.EventInfo {
	log.WithField("event-uri", eventURI).Debug("Getting detailed event info")
	et := m.MatchType(eventURI)
	if et == nil {
		return nil
	}

	// call Event Handler service to get event info
	handler := model.NewEventHandlerEndpoint(et.ServiceURL)
	info, err := handler.GetEventInfo(eventURI, secret)
	if err != nil {
		log.WithError(err).Error("Failed to get event info")
		return nil
	}

	return info
}
