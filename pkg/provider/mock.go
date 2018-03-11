package provider

import (
	"context"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/stretchr/testify/mock"
)

// NewEventProviderMock create EventProvider API mock
func NewEventProviderMock() *Mock {
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
func (c *Mock) GetEventInfo(ctx context.Context, event string, secret string) (*model.EventInfo, error) {
	args := c.Called(ctx, event, secret)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.EventInfo), args.Error(1)
}

// SubscribeToEvent mock
func (c *Mock) SubscribeToEvent(ctx context.Context, event, secret string, credentials map[string]string) (*model.EventInfo, error) {
	args := c.Called(ctx, event, secret, credentials)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.EventInfo), args.Error(1)
}

// UnsubscribeFromEvent mock
func (c *Mock) UnsubscribeFromEvent(ctx context.Context, event string, credentials map[string]string) error {
	args := c.Called(ctx, event, credentials)
	return args.Error(0)
}

// ConstructEventURI mock
func (c *Mock) ConstructEventURI(t string, k string, a string, values map[string]string) (string, error) {
	args := c.Called(t, k, a, values)
	return args.String(0), args.Error(1)
}
