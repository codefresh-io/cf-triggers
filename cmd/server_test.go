package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/controller"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestPingRoute(t *testing.T) {
	router := setupRouter(nil, nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "PONG", w.Body.String())
}

func TestHealthRoute(t *testing.T) {
	// prepare mocks
	pinger := new(model.MockPinger)
	codefresh := codefresh.NewCodefreshMockEndpoint()
	// setup router
	router := setupRouter(nil, nil, nil, nil, nil, pinger, codefresh)
	// setup mocks
	pinger.Mock.On("Ping").Return("PONG", nil)
	codefresh.On("Ping").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	pinger.Mock.AssertExpectations(t)
	codefresh.AssertExpectations(t)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "Healthy", w.Body.String())
}

func TestHealthRouteRedisError(t *testing.T) {
	// prepare mocks
	pinger := new(model.MockPinger)
	codefresh := codefresh.NewCodefreshMockEndpoint()
	// setup router
	router := setupRouter(nil, nil, nil, nil, nil, pinger, codefresh)
	// setup mocks
	pinger.On("Ping").Return("", errors.New("REDIS Error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	pinger.AssertExpectations(t)
	codefresh.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	expectedErr := controller.ErrorResult{
		Status:  http.StatusInternalServerError,
		Message: "failed to talk to backend",
		Error:   "REDIS Error",
	}
	var err controller.ErrorResult
	json.Unmarshal(w.Body.Bytes(), &err)

	assert.Equal(t, expectedErr, err)
}

func TestHealthRouteCodefreshError(t *testing.T) {
	// prepare mocks
	pinger := new(model.MockPinger)
	codefresh := codefresh.NewCodefreshMockEndpoint()
	// setup router
	router := setupRouter(nil, nil, nil, nil, nil, pinger, codefresh)
	// setup mocks
	pinger.On("Ping").Return("PONG", nil)
	codefresh.On("Ping").Return(errors.New("Codefresh Error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	pinger.AssertExpectations(t)
	codefresh.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	expectedErr := controller.ErrorResult{
		Status:  http.StatusInternalServerError,
		Message: "failed to talk to Codefresh API",
		Error:   "Codefresh Error",
	}
	var err controller.ErrorResult
	json.Unmarshal(w.Body.Bytes(), &err)

	assert.Equal(t, expectedErr, err)
}

func Test_ListTriggers(t *testing.T) {
	type args struct {
		account string
	}
	type expected struct {
		triggers []model.Trigger
	}
	tests := []struct {
		name        string
		args        args
		expected    expected
		errNotFound bool
		err         *controller.ErrorResult
	}{
		{
			name: "get public triggers",
			args: args{account: model.PublicAccount},
			expected: expected{
				[]model.Trigger{
					{
						Event:    "uri:test:" + model.PublicAccountHash,
						Pipeline: "pipeline1",
					},
					{
						Event:    "uri:test:" + model.PublicAccountHash,
						Pipeline: "pipeline2",
					},
					{
						Event:    "uri:test:event:" + model.PublicAccountHash,
						Pipeline: "pipeline3",
					},
				},
			},
		},
		{
			name: "internal error",
			args: args{account: model.PublicAccount},
			err: &controller.ErrorResult{
				Status:  http.StatusInternalServerError,
				Message: "failed to list triggers for event",
				Error:   "ERROR",
			},
		},
		{
			name:        "not found error",
			args:        args{account: model.PublicAccount},
			errNotFound: true,
			err: &controller.ErrorResult{
				Status:  http.StatusNotFound,
				Message: "failed to list triggers for event",
				Error:   "not found error",
			},
		},
	}
	for _, tt := range tests {
		// mock
		triggerReaderWriter := new(model.MockTriggerReaderWriter)
		// setup router
		router := setupRouter(nil, triggerReaderWriter, nil, nil, nil, nil, nil)
		// prepare mock
		call := triggerReaderWriter.On("GetEventTriggers", mock.Anything, "*")
		if tt.err != nil {
			if tt.errNotFound {
				call.Return(nil, model.ErrTriggerNotFound)
			} else {
				call.Return(nil, errors.New("ERROR"))
			}
		} else {
			call.Return(tt.expected.triggers, nil)
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/accounts/"+tt.args.account+"/triggers/", nil)
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			var got []model.Trigger
			json.Unmarshal(w.Body.Bytes(), &got)
			assert.Equal(t, tt.expected.triggers, got)
		} else {
			var err controller.ErrorResult
			json.Unmarshal(w.Body.Bytes(), &err)
		}

		// assert mock
		triggerReaderWriter.Mock.AssertExpectations(t)
	}
}
