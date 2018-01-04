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
		CheckPipelineExist(name, repoOwner, repoName string) error
		RunPipeline(repoOwner string, repoName string, name string, vars map[string]string) (string, error)
		Ping() error
	}

	// APIEndpoint Codefresh API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
	}
)

// ErrPipelineNotFound error when pipeline not found
var ErrPipelineNotFound = errors.New("codefresh: pipeline not found")

func checkResponse(text string, err error, status int) (string, error) {
	if err != nil {
		return "", err
	}
	if status < http.StatusOK || status >= http.StatusBadRequest {
		msg := fmt.Sprintf("%s - cf-api error: %s", text, http.StatusText(status))
		return "", fmt.Errorf(msg)
	}
	return "", nil
}

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
	log.Debugf("initializing cf-api %s ...", url)
	endpoint := sling.New().Base(url).Set("x-access-token", token).Set("Authorization", token)
	return &APIEndpoint{endpoint}
}

// find Codefresh pipeline by name and repo details (owner and name)
func (api *APIEndpoint) ping() error {
	resp, err := api.endpoint.New().Get("api/ping").ReceiveSuccess(nil)
	_, err = checkResponse("ping", err, resp.StatusCode)
	return err
}

// find Codefresh pipeline ID by repo (owner and name) and name
func (api *APIEndpoint) getPipelineID(repoOwner, repoName, name string) (string, error) {
	log.Debugf("getting pipeline repo-owner:%s repo-name:%s name:%s", repoOwner, repoName, name)
	// GET pipelines for repository
	type CFPipeline struct {
		ID   string `json:"_id"`
		Name string `json:"name"`
	}
	pipelines := new([]CFPipeline)
	resp, err := api.endpoint.New().Get(fmt.Sprint("api/services/", repoOwner, "/", repoName)).ReceiveSuccess(pipelines)
	_, err = checkResponse("get pipelines", err, resp.StatusCode)
	if err != nil {
		return "", err
	}

	// scan for pipeline ID
	for _, p := range *pipelines {
		if p.Name == name {
			log.Debugf("found id '%s' for the pipeline '%s'", p.ID, name)
			return p.ID, nil
		}
	}
	log.Errorf("failed to find '%s' pipeline", name)

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
		log.Errorf("failed to build run request for pipeline %s. error: %s", id, err)
		return "", err
	}

	// get run id
	resp, err := http.DefaultClient.Do(req)
	_, err = checkResponse("run pipeline", err, resp.StatusCode)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("close response body error")
		return "", err
	}
	runID := string(respData)

	if resp.StatusCode == http.StatusOK {
		log.Debugf("pipeline '%s' is running with run id: '%s'", id, runID)
	}
	return runID, nil
}

// CheckPipelineExist check if Codefresh pipeline exists
func (api *APIEndpoint) CheckPipelineExist(repoOwner, repoName, name string) error {
	_, err := api.getPipelineID(repoOwner, repoName, name)
	if err != nil {
		return err
	}
	return nil
}

// RunPipeline run Codefresh pipeline
func (api *APIEndpoint) RunPipeline(repoOwner, repoName, name string, vars map[string]string) (string, error) {
	// get pipeline id from repo and name
	id, err := api.getPipelineID(repoOwner, repoName, name)
	if err != nil && err != ErrPipelineNotFound {
		return "", err
	} else if err == ErrPipelineNotFound {
		log.Debugf("skipping pipeline '%s' for repository '%s/%s'", name, repoOwner, repoName)
		return "", nil
	}
	// invoke pipeline by id
	return api.runPipeline(id, vars)
}

// Ping Codefresh API
func (api *APIEndpoint) Ping() error {
	return api.ping()
}
