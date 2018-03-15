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
func (r *PipelineRunner) Run(account string, pipelines []string, vars map[string]string) ([]model.PipelineRun, error) {
	log.Debug("Running pipelines")
	var runs []model.PipelineRun
	for _, p := range pipelines {
		var pr model.PipelineRun
		log.WithField("pipeline", p).Debug("Running pipeline")
		pr.ID, pr.Error = r.pipelineSvc.RunPipeline(p, vars)
		if pr.Error != nil {
			log.WithError(pr.Error).Error("Failed to run pipeline")
		}
		log.WithFields(log.Fields{
			"run ID":    pr.ID,
			"run error": pr.Error,
		}).Debug("Pipeline run details")
		runs = append(runs, pr)
	}
	return runs, nil
}
