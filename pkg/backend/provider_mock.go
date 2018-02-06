package backend

import (
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/stretchr/testify/mock"
)

// NewEventProviderInformerMock create EventProviderInformer API mock
func NewEventProviderInformerMock() *Mock {
	return &Mock{}
}

// Mock mock
type Mock struct {
	mock.Mock
}

// GetTypes mock
func (c *Mock) GetTypes() []model.EventType {
	args := c.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]model.EventType)
}

// MatchType mock
func (c *Mock) MatchType(eventURI string) (*model.EventType, error) {
	args := c.Called(eventURI)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.EventType), args.Error(1)
}

// GetType mock
func (c *Mock) GetType(t string, k string) (*model.EventType, error) {
	args := c.Called(t, k)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.EventType), args.Error(1)
}

// GetEventInfo mock
func (c *Mock) GetEventInfo(eventURI string, secret string) (*model.EventInfo, error) {
	args := c.Called(eventURI, secret)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.EventInfo), args.Error(1)
}

// ConstructEventURI mock
func (c *Mock) ConstructEventURI(t string, k string, values map[string]string) (string, error) {
	args := c.Called(t, k, values)
	return args.String(0), args.Error(1)
}
