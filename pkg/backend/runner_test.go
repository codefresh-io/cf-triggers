package backend

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/codefresh-io/hermes/pkg/codefresh"

	"github.com/codefresh-io/hermes/pkg/model"
)

func TestNewRunner(t *testing.T) {
	mock := &codefresh.MockPipelineService{}
	tests := []struct {
		name string
		want model.Runner
	}{
		{
			"new runner",
			&PipelineRunner{mock},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRunner(mock); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRunner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipelineRunner_Run(t *testing.T) {
	type args struct {
		account   string
		pipelines []string
		vars      map[string]string
		event     model.NormalizedEvent
	}
	tests := []struct {
		name    string
		args    args
		want    []model.PipelineRun
		wantErr bool
	}{
		{
			"run pipeline",
			args{
				vars:      map[string]string{"V1": "AAA", "V2": "BBB"},
				pipelines: []string{"puid-1", "puid-2", "puid-3"},
				account:   "test",
				event:     model.NormalizedEvent{},
			},
			[]model.PipelineRun{
				{ID: "run1", Error: nil},
				{ID: "run2", Error: nil},
				{ID: "run3", Error: nil},
			},
			false,
		},
		{
			"run pipeline - some missing",
			args{
				vars:      map[string]string{"V1": "AAA", "V2": "BBB"},
				pipelines: []string{"puid-1", "puid-2", "puid-3"},
				account:   "test",
				event:     model.NormalizedEvent{},
			},
			[]model.PipelineRun{
				{ID: "run1", Error: nil},
				{ID: "", Error: fmt.Errorf("pipeline not found")},
				{ID: "run3", Error: nil},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &codefresh.MockPipelineService{}
			r := &PipelineRunner{
				pipelineSvc: mock,
			}
			for i, p := range tt.args.pipelines {
				mock.On("RunPipeline", tt.args.account, p, tt.args.vars, tt.args.event).Return(tt.want[i].ID, tt.want[i].Error)
			}
			got, err := r.Run(tt.args.account, tt.args.pipelines, tt.args.vars, tt.args.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("PipelineRunner.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PipelineRunner.Run() = %v, want %v", got, tt.want)
			}
			// assert expectation
			mock.AssertExpectations(t)
		})
	}
}
