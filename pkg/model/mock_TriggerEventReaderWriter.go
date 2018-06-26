// Code generated by mockery v1.0.0
package model

import context "context"
import mock "github.com/stretchr/testify/mock"

// MockTriggerEventReaderWriter is an autogenerated mock type for the TriggerEventReaderWriter type
type MockTriggerEventReaderWriter struct {
	mock.Mock
}

// CreateEvent provides a mock function with given fields: ctx, eventType, kind, secret, _a4, header, values
func (_m *MockTriggerEventReaderWriter) CreateEvent(ctx context.Context, eventType string, kind string, secret string, _a4 string, header string, values map[string]string) (*Event, error) {
	ret := _m.Called(ctx, eventType, kind, secret, _a4, header, values)

	var r0 *Event
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string, string, map[string]string) *Event); ok {
		r0 = rf(ctx, eventType, kind, secret, _a4, header, values)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Event)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, string, string, map[string]string) error); ok {
		r1 = rf(ctx, eventType, kind, secret, _a4, header, values)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteEvent provides a mock function with given fields: ctx, event, _a2
func (_m *MockTriggerEventReaderWriter) DeleteEvent(ctx context.Context, event string, _a2 string) error {
	ret := _m.Called(ctx, event, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, event, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetEvent provides a mock function with given fields: ctx, event
func (_m *MockTriggerEventReaderWriter) GetEvent(ctx context.Context, event string) (*Event, error) {
	ret := _m.Called(ctx, event)

	var r0 *Event
	if rf, ok := ret.Get(0).(func(context.Context, string) *Event); ok {
		r0 = rf(ctx, event)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Event)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, event)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEvents provides a mock function with given fields: ctx, eventType, kind, filter
func (_m *MockTriggerEventReaderWriter) GetEvents(ctx context.Context, eventType string, kind string, filter string) ([]Event, error) {
	ret := _m.Called(ctx, eventType, kind, filter)

	var r0 []Event
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) []Event); ok {
		r0 = rf(ctx, eventType, kind, filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Event)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string) error); ok {
		r1 = rf(ctx, eventType, kind, filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
