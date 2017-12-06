package backend

import (
	"errors"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	log "github.com/sirupsen/logrus"

	"github.com/codefresh-io/hermes/pkg/model"
)

// MemoryStore in memory trigger map store
type MemoryStore struct {
	triggers    map[string]model.Trigger
	pipelineSvc codefresh.PipelineService
}

// NewMemoryStore create new in memory trigger map store (for testing)
func NewMemoryStore(pipelineSvc codefresh.PipelineService) model.TriggerService {
	s := make(map[string]model.Trigger)
	return &MemoryStore{s, pipelineSvc}
}

// List list all triggers
func (m *MemoryStore) List(filter string) ([]model.Trigger, error) {
	triggers := []model.Trigger{}
	for _, v := range m.triggers {
		triggers = append(triggers, v)
	}
	return triggers, nil
}

// Get trigger by event URI
func (m *MemoryStore) Get(id string) (model.Trigger, error) {
	if trigger, ok := m.triggers[id]; ok {
		return trigger, nil
	}
	return model.EmptyTrigger, nil
}

// Add new trigger
func (m *MemoryStore) Add(t model.Trigger) error {
	m.triggers[t.Event] = t
	return nil
}

// Delete trigger
func (m *MemoryStore) Delete(id string) error {
	delete(m.triggers, id)
	return nil
}

// Update trigger
func (m *MemoryStore) Update(t model.Trigger) error {
	m.triggers[t.Event] = t
	return nil
}

// Run trigger pipelines
func (m *MemoryStore) Run(id string, vars map[string]string) error {
	if trigger, ok := m.triggers[id]; ok {
		for _, p := range trigger.Pipelines {
			err := m.pipelineSvc.RunPipeline(p.Name, p.RepoOwner, p.RepoName, vars)
			if err != nil {
				log.Error(err)
				return err
			}
		}
		return nil
	}
	return nil
}

// CheckSecret check trigger secret
func (m *MemoryStore) CheckSecret(id string, secret string) error {
	if trigger, ok := m.triggers[id]; ok {
		if trigger.Secret != secret {
			return errors.New("invalid secret")
		}
	}
	return nil
}
