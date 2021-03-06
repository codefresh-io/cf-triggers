// Code generated by mockery v1.0.0
package model

import mock "github.com/stretchr/testify/mock"

// MockPinger is an autogenerated mock type for the Pinger type
type MockPinger struct {
	mock.Mock
}

// Ping provides a mock function with given fields:
func (_m *MockPinger) Ping() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
