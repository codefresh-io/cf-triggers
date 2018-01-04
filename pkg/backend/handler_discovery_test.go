package backend

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/codefresh-io/hermes/pkg/model"
	log "github.com/sirupsen/logrus"
)

const (
	testTypesFilename = "test_types.json"
)

var (
	types = model.EventTypes{
		Types: []model.EventType{
			model.EventType{
				Type:        "registry",
				Kind:        "dockerhub",
				ServiceURL:  "http://service:8080",
				URITemplate: "index.docker.io:{{repo-owner}}:{{repo-name}}:push",
				URIPattern:  `^index\.docker\.io:[a-z0-9_-]+:[a-z0-9_-]+:push$`,
				Config: []model.ConfigField{
					{Name: "repo-owner", Type: "string", Validator: "^[A-z]+$", Required: true},
					{Name: "repo-name", Type: "string", Validator: "^[A-z]+$", Required: true},
				},
			},
		},
	}
)

// helper: create valid config file
func createValidConfig(prefix string) string {
	tmpfile, err := ioutil.TempFile("", prefix+"_"+testTypesFilename)
	if err != nil {
		log.Fatal(err)
	}

	content, err := json.Marshal(types)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	return tmpfile.Name()
}

func createInvalidConfig() string {
	tmpfile, err := ioutil.TempFile("", testTypesFilename)
	if err != nil {
		log.Fatal(err)
	}

	content := []byte("Non JSON file. Should lead to error!")
	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	return tmpfile.Name()
}

func Test_NewEventHandlerManagerSingleton(t *testing.T) {
	// create valid config file
	config := createValidConfig("singleton")
	defer os.Remove(config)
	// create 2 instances
	manager1 := NewEventHandlerManager(config, true)
	manager2 := NewEventHandlerManager(config, true)
	if manager1 != manager2 {
		t.Error("non singleton EventHandlerManager")
	}
	manager1.Close()
}

func Test_NewEventHandlerManagerInvalid(t *testing.T) {
	// create invalid config file
	config := createInvalidConfig()
	defer os.Remove(config)
	// create manager; ignores file error
	manager := newTestEventHandlerManager(config)
	manager.Close()
}

func Test_loadInvalidConfig(t *testing.T) {
	// create invalid config file
	config := createInvalidConfig()
	defer os.Remove(config)
	// create manager; ignores file error
	manager := newTestEventHandlerManager(config)
	defer manager.Close()
	// try to load config explicitly
	types, err := loadEventHandlerTypes(config)
	if err == nil {
		t.Error("should fail loading types from invalid file")
	}
	if types.Types != nil {
		t.Error("types should be nil on error")
	}
}

func Test_loadNonExistingConfig(t *testing.T) {
	// create invalid config file
	config := "non-existing.file.json"
	defer os.Remove(config)
	// create manager; ignores file error
	manager := newTestEventHandlerManager(config)
	defer manager.Close()
	// try to load config explicitly
	types, err := loadEventHandlerTypes(config)
	if err == nil {
		t.Error("should fail loading types from non-existing file")
	}
	if types.Types != nil {
		t.Error("types should be nil on error")
	}
}

func Test_loadValidConfig(t *testing.T) {
	// create valid config file
	config := createValidConfig("valid")
	defer os.Remove(config)
	// create manager; ignores file error
	manager := newTestEventHandlerManager(config)
	defer manager.Close()
	// try to load config explicitly
	types, err := loadEventHandlerTypes(config)
	if err != nil {
		t.Errorf("unexpected error loading types from file %v", err)
	}
	if types.Types == nil && len(types.Types) != 1 {
		t.Error("types should have one properly defined type")
	}
}

func Test_monitorConfigFile(t *testing.T) {
	// create valid config file
	config := createValidConfig("monitor")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventHandlerManager(config)
	defer manager.Close()

	// update config file
	newType := model.EventType{Type: "new-type", ServiceURL: "http://new-service"}
	updatedTypes := types
	updatedTypes.Types = append(updatedTypes.Types, newType)

	// write to file
	content, err := json.Marshal(updatedTypes)
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(config, content, 0644); err != nil {
		log.Fatal(err)
	}

	// get types, after 2 sec sleep
	time.Sleep(2 * time.Second)
	checkTypes := manager.GetTypes()
	if len(checkTypes) != 2 {
		t.Error("failed to update types")
	}
}

func Test_GetType(t *testing.T) {
	// create valid config file
	config := createValidConfig("get_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventHandlerManager(config)
	defer manager.Close()
	// get type
	eventType := manager.GetType("registry", "dockerhub")
	if eventType == nil {
		t.Error("fail to get type from valid configuration")
	}
}

func Test_NonExistingGetType(t *testing.T) {
	// create valid config file
	config := createValidConfig("get_ne_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventHandlerManager(config)
	defer manager.Close()
	// get type
	eventType := manager.GetType("registry", "non-existing")
	if eventType != nil {
		t.Error("non-nil type from configuration")
	}
}

func TestEventHandlerManager_MatchExistingType(t *testing.T) {
	// create valid config file
	config := createValidConfig("match_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventHandlerManager(config)
	defer manager.Close()
	// match type
	eventType := manager.MatchType("index.docker.io:codefresh:fortune:push")
	if eventType == nil {
		t.Error("failed to find type by uri")
	}
}

func TestEventHandlerManager_MatchTypeNil(t *testing.T) {
	// create valid config file
	config := createValidConfig("match_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventHandlerManager(config)
	defer manager.Close()
	// match type
	eventType := manager.MatchType("index.docker.io:not-valid:push")
	if eventType != nil {
		t.Error("non-nil type for invalid uri")
	}
}
