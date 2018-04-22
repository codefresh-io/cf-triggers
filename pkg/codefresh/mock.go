package codefresh

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// NewCodefreshMockEndpoint create Codefresh API mock
func NewCodefreshMockEndpoint() *Mock {
	return &Mock{}
}

// Mock mock CF API
type Mock struct {
	mock.Mock
}

// GetPipeline mock
func (c *Mock) GetPipeline(ctx context.Context, account, id string) (*Pipeline, error) {
	args := c.Called(ctx, account, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Pipeline), args.Error(1)
}

// RunPipeline mock
func (c *Mock) RunPipeline(accountID string, id string, vars map[string]string) (string, error) {
	args := c.Called(accountID, id, vars)
	return args.String(0), args.Error(1)
}

// Ping mock
func (c *Mock) Ping() error {
	args := c.Called()
	return args.Error(0)
}
