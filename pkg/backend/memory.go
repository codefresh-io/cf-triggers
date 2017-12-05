package backend

import (
	"github.com/codefresh-io/cf-triggers/pkg/model"
)

// MemoryStore in memory trigger map store
type MemoryStore struct {
	triggers map[string]model.Trigger
}

// NewMemoryStore create new in memory trigger map store (for testing)
func NewMemoryStore() model.TriggerService {
	s := make(map[string]model.Trigger)
	return &MemoryStore{s}
}

// List list all triggers
func (m *MemoryStore) List(filter string) ([]model.Trigger, error) {
	var triggers []model.Trigger
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
	return model.Trigger{}, nil
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
