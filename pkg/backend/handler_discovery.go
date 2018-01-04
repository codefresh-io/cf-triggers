package backend

import (
	"io/ioutil"
	"regexp"
	"sync"
	"time"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
	yaml "gopkg.in/yaml.v2"

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
		var err error
		instance.eventTypes, err = loadEventHandlerTypes(configFile)
		if err != nil {
			log.Error(err)
		}
		// start monitoring
		if !skipMonitor {
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
		log.Errorf("failed to read config file %v", err)
		return eventTypes, err
	}

	err = yaml.Unmarshal(eventTypesData, &eventTypes)
	if err != nil {
		log.Errorf("bad config file format %v", err)
		return eventTypes, err
	}
	return eventTypes, nil
}

// Close - free file watcher resources
func (m *EventHandlerManager) Close() {
	if m.watcher != nil {
		m.watcher.Close()
	}
}

// NOTE: should be called only once
// monitor configuration directory to discover new/updated/deleted Event Handlers
func (m *EventHandlerManager) monitorConfigFile() *util.FileWatcher {
	// Watch the file for modification and update the config manager with the new config when it's available
	watcher, err := util.WatchFile(m.configFile, time.Second, func() {
		log.Debug("Event Handler types file updated")
		m.Lock()
		defer m.Unlock()
		var err error
		m.eventTypes, err = loadEventHandlerTypes(m.configFile)
		if err != nil {
			log.Error(err)
		}
	})
	if err != nil {
		log.Error(err)
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

	log.Error("fail to fetch event types")
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

	log.Errorf("fail to find event type %s kind %s", eventType, eventKind)
	return nil
}

// MatchType match event type by uri
func (m *EventHandlerManager) MatchType(eventURI string) *model.EventType {
	m.Lock()
	defer m.Unlock()

	for _, e := range m.eventTypes.Types {
		r, err := regexp.Compile(e.URIPattern)
		if err != nil {
			log.Errorf("bad uri regex pattern for type:%s", e.Type)
			continue // skip
		}
		if r.MatchString(eventURI) {
			return &e
		}
	}

	log.Errorf("fail to find event type for %s", eventURI)
	return nil
}

// GetEventInfo get individual event handler type (by type and kind)
func (m *EventHandlerManager) GetEventInfo(eventURI string, secret string) *model.EventInfo {
	et := m.MatchType(eventURI)
	if et == nil {
		return nil
	}

	// call Event Handler service to get event info
	handler := model.NewEventHandlerEndpoint(et.ServiceURL)
	info, err := handler.GetEventInfo(eventURI, secret)
	if err != nil {
		log.Error(err)
		return nil
	}

	return info
}
