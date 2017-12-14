package codefresh

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// PipelineService Codefresh Service
	PipelineService interface {
		RunPipeline(name string, repoOwner string, repoName string, vars map[string]string) (string, error)
	}

	// APIEndpoint Codefresh API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
	}
)

// ErrPipelineNotFound error when pipeline not found
var ErrPipelineNotFound = errors.New("codefresh: pipeline not found")

// create a new map of variables
// each key is converted to UPPER case and prefixed with 'EVENT_'
func preprocessVariables(vars map[string]string) map[string]string {
	newVars := make(map[string]string)
	for k, v := range vars {
		k = strings.ToUpper(k)
		if !strings.HasPrefix(k, "EVENT_") {
			k = "EVENT_" + k
		}
		newVars[k] = v
	}
	return newVars
}

// NewCodefreshEndpoint create new Codefresh API endpoint from url and API token
func NewCodefreshEndpoint(url, token string) PipelineService {
	endpoint := sling.New().Base(url).Set("x-access-token", token)
	return &APIEndpoint{endpoint}
}

// find Codefresh pipeline by name and repo details (owner and name)
func (api *APIEndpoint) getPipelineByNameAndRepo(name, repoOwner, repoName string) (string, error) {
	// GET pipelines for repository
	type CFPipeline struct {
		ID   string `json:"_id"`
		Name string `json:"name"`
	}
	pipelines := new([]CFPipeline)
	if _, err := api.endpoint.New().Get(fmt.Sprint("api/services/", repoOwner, "/", repoName)).ReceiveSuccess(pipelines); err != nil {
		log.Error(err)
		return "", ErrPipelineNotFound
	}

	// scan for pipeline ID
	for _, p := range *pipelines {
		if p.Name == name {
			log.Debugf("Found id '%s' for the pipeline '%s'", p.ID, name)
			return p.ID, nil
		}
	}
	log.Errorf("Failed to find '%s' pipeline", name)

	return "", ErrPipelineNotFound
}

// run Codefresh pipeline
func (api *APIEndpoint) runPipeline(id string, vars map[string]string) (string, error) {
	log.Debugf("Going to run pipeline id: %s", id)
	type BuildRequest struct {
		Branch    string            `json:"branch,omitempty"`
		Variables map[string]string `json:"variables,omitempty"`
	}

	// start new run
	body := &BuildRequest{
		Branch:    "master",
		Variables: preprocessVariables(vars),
	}
	req, err := api.endpoint.New().Post(fmt.Sprint("api/builds/", id)).BodyJSON(body).Request()
	if err != nil {
		log.Error(err)
		return "", err
	}

	// get run id
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
		return "", err
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return "", err
	}
	runID := string(respData)

	if resp.StatusCode == http.StatusOK {
		log.Debugf("Pipeline '%s' is running with run id: '%s'", id, runID)
	}
	return runID, nil
}

// RunPipeline run Codefresh pipeline
func (api *APIEndpoint) RunPipeline(name, repoOwner, repoName string, vars map[string]string) (string, error) {
	// get pipeline id from repo and name
	id, err := api.getPipelineByNameAndRepo(name, repoOwner, repoName)
	if err != nil && err != ErrPipelineNotFound {
		log.Error(err)
		return "", err
	} else if err == ErrPipelineNotFound {
		log.Debugf("Skipping pipeline '%s' for repository '%s/%s'", name, repoOwner, repoName)
		return "", nil
	}
	// invoke pipeline by id
	return api.runPipeline(id, vars)
}
