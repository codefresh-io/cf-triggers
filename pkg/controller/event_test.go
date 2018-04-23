package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTriggerEventController_GetEvent(t *testing.T) {
	type args struct {
		eventURI        string
		eventURIEncoded string
	}
	tests := []struct {
		name     string
		args     args
		event    *model.Event
		wantErr  error
		wantCode int
	}{
		{
			name: "get event",
			args: args{
				eventURI:        "cron:codefresh:0 0/30 10-13 ? * WED,FR:account",
				eventURIEncoded: "cron:codefresh:0%200%2F30%2010-13%20%3F%20%2A%20WED%2CFR:account",
			},
			event: &model.Event{
				Type: "cron",
				Kind: "codefresh",
			},
		},
		{
			name: "get event with error",
			args: args{
				eventURI:        "cron:codefresh:0 0/30 10-13 ? * WED,FR:account",
				eventURIEncoded: "cron:codefresh:0%200%2F30%2010-13%20%3F%20%2A%20WED%2CFR:account",
			},
			wantErr:  errors.New("TEST ERROR"),
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "get event with NotFound error",
			args: args{
				eventURI:        "cron:codefresh:0 0/30 10-13 ? * WED,FR:account",
				eventURIEncoded: "cron:codefresh:0%200%2F30%2010-13%20%3F%20%2A%20WED%2CFR:account",
			},
			wantErr:  model.ErrTriggerNotFound,
			wantCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &model.MockTriggerEventReaderWriter{}
			c := &TriggerEventController{
				svc: mockSvc,
			}
			w := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(w)
			ginCtx.Params = gin.Params{gin.Param{Key: "event", Value: tt.args.eventURIEncoded}}
			ginCtx.Request, _ = http.NewRequest("GET", "/test", nil)
			// prepare mock
			call := mockSvc.On("GetEvent", getContext(ginCtx), tt.args.eventURI)
			call.Return(tt.event, tt.wantErr)
			// invoke
			c.GetEvent(ginCtx)
			t.Log(w.Body.String())
			// assert code
			// assert code
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantCode, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}
			// assert exectations
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestTriggerEventController_GetEvents(t *testing.T) {
	type fields struct {
		svc model.TriggerEventReaderWriter
	}
	type args struct {
		typeStr string
		kind    string
		filter  string
		public  bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		events   []model.Event
		wantErr  error
		wantCode int
	}{
		{
			name: "get events",
			args: args{
				typeStr: "test-type",
				kind:    "test-kind",
				filter:  "test-filter",
			},
			events: []model.Event{
				{
					URI:  "uri:1",
					Type: "test-type",
					Kind: "test-kind",
				},
				{
					URI:  "uri:2",
					Type: "test-type",
					Kind: "test-kind",
				},
			},
		},
		{
			name: "get public events",
			args: args{
				typeStr: "test-type",
				kind:    "test-kind",
				filter:  "test-filter",
				public:  true,
			},
			events: []model.Event{
				{
					URI:  "uri:1",
					Type: "test-type",
					Kind: "test-kind",
				},
				{
					URI:  "uri:2",
					Type: "test-type",
					Kind: "test-kind",
				},
			},
		},
		{
			name: "get events with error",
			args: args{
				typeStr: "test-type",
				kind:    "test-kind",
				filter:  "test-filter",
			},
			wantErr:  errors.New("TEST ERROR"),
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "get events with NotFound error",
			args: args{
				typeStr: "test-type",
				kind:    "test-kind",
				filter:  "test-filter",
			},
			wantErr:  model.ErrTriggerNotFound,
			wantCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		mockSvc := &model.MockTriggerEventReaderWriter{}
		t.Run(tt.name, func(t *testing.T) {
			c := &TriggerEventController{
				svc: mockSvc,
			}
			w := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(w)
			path := fmt.Sprintf("/test?type=%s&kind=%s&filter=%s&public=%v", tt.args.typeStr, tt.args.kind, tt.args.filter, tt.args.public)
			ginCtx.Request, _ = http.NewRequest("GET", path, nil)
			// prepare mock
			call := mockSvc.On("GetEvents", mock.Anything, tt.args.typeStr, tt.args.kind, tt.args.filter)
			call.Return(tt.events, tt.wantErr)
			// invoke
			c.GetEvents(ginCtx)
			t.Log(w.Body.String())
			// assert code
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantCode, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}
			// assert exectations
			mockSvc.AssertExpectations(t)
		})
	}
}
