package provider

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
				URITemplate: "registry:dockerhub:{{namespace}}:{{name}}:push",
				URIPattern:  `^registry:dockerhub:[a-z0-9_-]+:[a-z0-9_-]+:push$`,
				Config: []model.ConfigField{
					{Name: "namespace", Type: "string", Validator: "^[a-z0-9_-]+$", Required: true},
					{Name: "name", Type: "string", Validator: "^[a-z0-9_-]+$", Required: true},
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

func Test_NewEventProviderManagerSingleton(t *testing.T) {
	// create valid config file
	config := createValidConfig("singleton")
	defer os.Remove(config)
	// create 2 instances
	manager1 := NewEventProviderManager(config, true)
	manager2 := NewEventProviderManager(config, true)
	if manager1 != manager2 {
		t.Error("non singleton EventProviderManager")
	}
	manager1.Close()
}

func Test_NewEventProviderManagerInvalid(t *testing.T) {
	// create invalid config file
	config := createInvalidConfig()
	defer os.Remove(config)
	// create manager; ignores file error
	manager := newTestEventProviderManager(config)
	manager.Close()
}

func Test_loadInvalidConfig(t *testing.T) {
	// create invalid config file
	config := createInvalidConfig()
	defer os.Remove(config)
	// create manager; ignores file error
	manager := newTestEventProviderManager(config)
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
	manager := newTestEventProviderManager(config)
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
	manager := newTestEventProviderManager(config)
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
	manager := newTestEventProviderManager(config)
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
	manager := newTestEventProviderManager(config)
	defer manager.Close()
	// get type
	_, err := manager.GetType("registry", "dockerhub")
	if err != nil {
		t.Error("fail to get type from valid configuration")
	}
}

func Test_NonExistingGetType(t *testing.T) {
	// create valid config file
	config := createValidConfig("get_ne_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventProviderManager(config)
	defer manager.Close()
	// get type
	_, err := manager.GetType("registry", "non-existing")
	if err == nil {
		t.Error("non-nil type from configuration")
	}
}

func TestEventProviderManager_MatchExistingType(t *testing.T) {
	// create valid config file
	config := createValidConfig("match_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventProviderManager(config)
	defer manager.Close()
	// match type
	_, err := manager.MatchType("registry:dockerhub:codefresh:fortune:push:" + model.CalculateAccountHash("A"))
	if err != nil {
		t.Error("failed to find type by uri")
	}
}

func TestEventProviderManager_MatchExistingTypeAccount(t *testing.T) {
	// create valid config file
	config := createValidConfig("match_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventProviderManager(config)
	defer manager.Close()
	// match type
	_, err := manager.MatchType("registry:dockerhub:codefresh:fortune:push:cb1e73c5215b")
	if err != nil {
		t.Error("failed to find type by uri")
	}
}

func TestEventProviderManager_MatchTypeNil(t *testing.T) {
	// create valid config file
	config := createValidConfig("match_type")
	defer os.Remove(config)
	// create manager; and start monitoring
	manager := newTestEventProviderManager(config)
	defer manager.Close()
	// match type
	_, err := manager.MatchType("registry:dockerhub:not-valid:push")
	if err == nil {
		t.Error("non-nil type for invalid uri")
	}
}

func TestEventProviderManager_ConstructEventURI(t *testing.T) {
	type args struct {
		t      string
		k      string
		a      string
		values map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"happy path",
			args{
				t:      "registry",
				k:      "dockerhub",
				a:      model.PublicAccount,
				values: map[string]string{"namespace": "codefresh", "name": "fortune"},
			},
			"registry:dockerhub:codefresh:fortune:push:" + model.PublicAccountHash,
			false,
		},
		{
			"happy path with account",
			args{
				t:      "registry",
				k:      "dockerhub",
				a:      "5672d8deb6724b6e359adf62",
				values: map[string]string{"namespace": "codefresh", "name": "fortune"},
			},
			"registry:dockerhub:codefresh:fortune:push:cb1e73c5215b",
			false,
		},
		{
			"non existing type",
			args{
				t:      "non-existing-type",
				k:      "any",
				values: map[string]string{"namespace": "codefresh", "name": "fortune"},
			},
			"",
			true,
		},
		{
			"fail value validation",
			args{
				t:      "registry",
				k:      "dockerhub",
				values: map[string]string{"namespace": "codefresh!", "name": "fortune@"},
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create valid config file
			config := createValidConfig("get_type")
			defer os.Remove(config)
			// create manager; and start monitoring
			manager := newTestEventProviderManager(config)
			defer manager.Close()
			got, err := manager.ConstructEventURI(tt.args.t, tt.args.k, tt.args.a, tt.args.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("EventProviderManager.ConstructEventURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EventProviderManager.ConstructEventURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
