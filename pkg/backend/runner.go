package backend

import (
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	log "github.com/sirupsen/logrus"
)

// PipelineRunner runs Codefresh pipelines
type PipelineRunner struct {
	pipelineSvc codefresh.PipelineService
}

// NewRunner initialize new PipelineRunner
func NewRunner(ps codefresh.PipelineService) model.Runner {
	return &PipelineRunner{ps}
}

// Run Codefresh pipelines: return arrays of runs and errors
func (r *PipelineRunner) Run(pipelines []model.Pipeline, vars map[string]string) ([]model.PipelineRun, error) {
	var runs []model.PipelineRun
	for _, p := range pipelines {
		var pr model.PipelineRun
		pr.ID, pr.Error = r.pipelineSvc.RunPipeline(p.RepoOwner, p.RepoName, p.Name, vars)
		if pr.Error != nil {
			log.Error(pr.Error)
		}
		runs = append(runs, pr)
	}
	return runs, nil
}
