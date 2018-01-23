package codefresh

import (
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

// CheckPipelineExists mock
func (c *Mock) CheckPipelineExists(pipelineUID string) (bool, error) {
	args := c.Called(pipelineUID)
	return args.Bool(0), args.Error(1)
}

// RunPipeline mock
func (c *Mock) RunPipeline(pipelineUID string, vars map[string]string) (string, error) {
	args := c.Called(pipelineUID, vars)
	return args.String(0), args.Error(1)
}

// Ping mock
func (c *Mock) Ping() error {
	args := c.Called()
	return args.Error(0)
}
