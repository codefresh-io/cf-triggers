package backend

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/codefresh-io/hermes/pkg/codefresh"

	"github.com/codefresh-io/hermes/pkg/model"
)

func TestNewRunner(t *testing.T) {
	mock := codefresh.NewCodefreshMockEndpoint()
	type args struct {
		mock *codefresh.Mock
	}
	tests := []struct {
		name string
		args args
		want model.Runner
	}{
		{
			"new runner",
			args{mock},
			&PipelineRunner{mock},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRunner(tt.args.mock); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRunner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipelineRunner_Run(t *testing.T) {
	type fields struct {
		mock *codefresh.Mock
	}
	type args struct {
		account   string
		pipelines []string
		vars      map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []model.PipelineRun
		wantErr bool
	}{
		{
			"run pipeline",
			fields{codefresh.NewCodefreshMockEndpoint()},
			args{
				vars:      map[string]string{"V1": "AAA", "V2": "BBB"},
				pipelines: []string{"puid-1", "puid-2", "puid-3"},
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
			fields{codefresh.NewCodefreshMockEndpoint()},
			args{
				vars:      map[string]string{"V1": "AAA", "V2": "BBB"},
				pipelines: []string{"puid-1", "puid-2", "puid-3"},
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
			r := &PipelineRunner{
				pipelineSvc: tt.fields.mock,
			}
			for i, p := range tt.args.pipelines {
				tt.fields.mock.On("RunPipeline", p, tt.args.vars).Return(tt.want[i].ID, tt.want[i].Error)
			}
			got, err := r.Run(tt.args.account, tt.args.pipelines, tt.args.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("PipelineRunner.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PipelineRunner.Run() = %v, want %v", got, tt.want)
			}
			// assert expectation
			tt.fields.mock.AssertExpectations(t)
		})
	}
}
