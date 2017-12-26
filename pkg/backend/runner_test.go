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
		pipelines []model.Pipeline
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
				vars: map[string]string{"V1": "AAA", "V2": "BBB"},
				pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline2"},
					{RepoOwner: "ownerA", RepoName: "repoB", Name: "pipeline3"},
				},
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
				vars: map[string]string{"V1": "AAA", "V2": "BBB"},
				pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline2"},
					{RepoOwner: "ownerA", RepoName: "repoB", Name: "pipeline3"},
				},
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
				tt.fields.mock.On("RunPipeline", p.RepoOwner, p.RepoName, p.Name, tt.args.vars).Return(tt.want[i].ID, tt.want[i].Error)
			}
			got, err := r.Run(tt.args.pipelines, tt.args.vars)
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
