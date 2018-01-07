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

// CheckPipelineExist mock
func (c *Mock) CheckPipelineExist(account, repoOwner string, repoName string, name string) error {
	args := c.Called(account, repoOwner, repoName, name)
	return args.Error(0)
}

// RunPipeline mock
func (c *Mock) RunPipeline(account string, repoOwner string, repoName string, name string, vars map[string]string) (string, error) {
	args := c.Called(account, repoOwner, repoName, name, vars)
	return args.String(0), args.Error(1)
}

// Ping mock
func (c *Mock) Ping() error {
	args := c.Called()
	return args.Error(0)
}
