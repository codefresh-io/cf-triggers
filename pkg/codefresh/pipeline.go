package codefresh

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"

	"github.com/codefresh-io/hermes/pkg/version"
)

type (
	// PipelineService Codefresh Service
	PipelineService interface {
		CheckPipelineExists(pipelineUID string) (bool, error)
		RunPipeline(pipelineUID string, vars map[string]string) (string, error)
		Ping() error
	}

	// APIEndpoint Codefresh API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
		internal bool
	}
)

// ErrPipelineNotFound error when pipeline not found
var ErrPipelineNotFound = errors.New("codefresh: pipeline not found")

func checkResponse(text string, err error, resp *http.Response) error {
	if err != nil {
		return err
	}
	if resp != nil && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest) {
		msg := fmt.Sprintf("%s cfapi error: %s", text, http.StatusText(resp.StatusCode))
		// try to get details from response body
		defer resp.Body.Close()
		respData, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			details := string(respData)
			msg = fmt.Sprintf("%s; more details: %s", msg, details)
		}
		return fmt.Errorf(msg)
	}
	return nil
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
	log.WithFields(log.Fields{
		"url":   url,
		"token": fmt.Sprint("HIDDEN..", token[len(token)-6:]),
	}).Debug("initializing cf-api")
	endpoint := sling.New().Base(url).Set("Authorization", token).Set("User-Agent", version.UserAgent)
	return &APIEndpoint{endpoint, token == ""}
}

// find Codefresh pipeline by name and repo details (owner and name)
func (api *APIEndpoint) ping() error {
	resp, err := api.endpoint.New().Get("api/ping").ReceiveSuccess(nil)
	return checkResponse("ping", err, resp)
}

// find Codefresh pipeline by name and repo details (owner and name)
func (api *APIEndpoint) checkPipelineExists(id string) (bool, error) {
	log.Debugf("getting pipeline %s", id)
	// GET pipelines for repository
	type CFPipeline struct {
		ID string `json:"_id"`
	}
	pipeline := new(CFPipeline)
	var resp *http.Response
	var err error
	if api.internal {
		// use internal cfapi - another endpoint and need to ass account
		log.Debug("using internal cfapi")
		resp, err = api.endpoint.New().Get(fmt.Sprint("api/pipelines/", id)).ReceiveSuccess(pipeline)
	} else {
		// use public cfapi
		log.Debug("using public cfapi")
		resp, err = api.endpoint.New().Get(fmt.Sprint("api/pipelines/", id)).ReceiveSuccess(pipeline)
	}
	err = checkResponse("get pipelines", err, resp)
	if err != nil {
		return false, err
	}

	// scan for pipeline ID
	if pipeline != nil {
		log.WithField("pipeline-uid", id).Debug("found pipeline by id")
		return true, nil
	}

	log.WithField("pipeline-uid", id).Error("failed to find pipeline with id")
	return false, ErrPipelineNotFound
}

// run Codefresh pipeline
func (api *APIEndpoint) runPipeline(id string, vars map[string]string) (string, error) {
	log.WithField("pipeline-uid", id).Debug("Going to run pipeline")
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
		log.WithFields(log.Fields{
			"pipeline-uid": id,
			"error":        err,
		}).Error("failed to build run request for pipeline")
		return "", err
	}

	// get run id
	resp, err := http.DefaultClient.Do(req)
	err = checkResponse("run pipeline", err, resp)
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
		log.WithFields(log.Fields{
			"pipeline-uid": id,
			"run-id":       runID,
		}).Debug("pipeline is running")
	}
	return runID, nil
}

// CheckPipelineExists check pipeline exists
func (api *APIEndpoint) CheckPipelineExists(pipelineUID string) (bool, error) {
	// invoke pipeline by id
	return api.checkPipelineExists(pipelineUID)
}

// RunPipeline run Codefresh pipeline
func (api *APIEndpoint) RunPipeline(pipelineUID string, vars map[string]string) (string, error) {
	// invoke pipeline by id
	return api.runPipeline(pipelineUID, vars)
}

// Ping Codefresh API
func (api *APIEndpoint) Ping() error {
	return api.ping()
}
