package codefresh

import (
	"fmt"
	"net/http"

	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// PipelineService Codefresh Service
	PipelineService interface {
		RunPipeline(name string, repoOwner string, repoName string, vars map[string]string) error
	}

	// APIEndpoint Codefresh API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
	}
)

// NewCodefreshEndpoint create new Codefresh API endpoint from url and API token
func NewCodefreshEndpoint(url, token string) PipelineService {
	endpoint := sling.New().Base(url).Set("x-access-token", token)
	return &APIEndpoint{endpoint}
}

func (api *APIEndpoint) getPipelineByNameAndRepo(name, repoOwner, repoName string) (string, error) {
	// GET pipelines for repository
	type CFPipeline struct {
		ID   string `json:"_id"`
		Name string `json:"name"`
	}
	pipelines := new([]CFPipeline)
	if _, err := api.endpoint.New().Get(fmt.Sprint("api/services/", repoOwner, "/", repoName)).ReceiveSuccess(pipelines); err != nil {
		log.Error(err)
		return "", err
	}

	// scan for pipeline ID
	for _, p := range *pipelines {
		if p.Name == name {
			log.Debugf("Found id '%s' for the pipeline '%s'", p.ID, name)
			return p.ID, nil
		}
	}

	return "", fmt.Errorf("Failed to find '%s' pipeline", name)
}

func (api *APIEndpoint) runPipeline(id string, vars map[string]string) error {
	log.Debugf("Going to run pipeline id: %s", id)
	type BuildRequest struct {
		Branch    string            `json:"branch,omitempty"`
		Variables map[string]string `json:"variables,omitempty"`
	}

	body := &BuildRequest{
		Branch:    "master",
		Variables: vars,
	}
	resp, err := api.endpoint.New().Post(fmt.Sprint("api/builds/", id)).BodyJSON(body).ReceiveSuccess(nil)
	if err != nil {
		log.Error(err)
		return err
	}
	if resp.StatusCode == http.StatusOK {
		log.Debugf("Pipeline '%s' is running...", id)
	}
	return nil
}

// RunPipeline run Codefresh pipeline
func (api *APIEndpoint) RunPipeline(name, repoOwner, repoName string, vars map[string]string) error {
	// get pipeline id from repo and name
	id, err := api.getPipelineByNameAndRepo(name, repoOwner, repoName)
	if err != nil {
		log.Error(err)
		return err
	}
	// invoke pipeline by id
	return api.runPipeline(id, vars)
}
