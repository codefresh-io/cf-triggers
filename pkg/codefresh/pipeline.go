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
	// Pipeline codefresh pipeline identifier
	Pipeline struct {
		ID      string
		Account string
	}

	// PipelineService Codefresh Service
	PipelineService interface {
		GetPipeline(account, id string) (*Pipeline, error)
		RunPipeline(id string, vars map[string]string) (string, error)
		Ping() error
	}

	// APIEndpoint Codefresh API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
		internal bool
	}
)

var (
	// RequestID request ID for logging
	RequestID = "X-Request-Id"
	// AuthEntity Codefresh authenticated entity JSON
	AuthEntity = "X-Authenticated-Entity-Json"
)

// ErrPipelineNotFound error when pipeline not found
var ErrPipelineNotFound = errors.New("codefresh: pipeline not found")

// ErrPipelineNoMatch error when pipeline not found
var ErrPipelineNoMatch = errors.New("codefresh: pipeline account does not match")

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
	if len(token) > 6 {
		log.WithFields(log.Fields{
			"url":          url,
			"internal-api": false,
			"token":        fmt.Sprint("HIDDEN..", token[len(token)-6:]),
		}).Debug("initializing cf-api")
	} else {
		log.WithFields(log.Fields{
			"url":          url,
			"internal-api": true,
		}).Debug("initializing cf-api")
	}
	endpoint := sling.New().Base(url).Set("Authorization", token).Set("User-Agent", version.UserAgent)
	return &APIEndpoint{endpoint, token == ""}
}

// find Codefresh pipeline by name and repo details (owner and name)
func (api *APIEndpoint) ping() error {
	resp, err := api.endpoint.New().Get("api/ping").ReceiveSuccess(nil)
	return checkResponse("ping", err, resp)
}

// find Codefresh pipeline by name and repo details (owner and name)
func (api *APIEndpoint) getPipeline(account, id string) (*Pipeline, error) {
	log.WithField("pipeline", id).Debug("getting pipeline")
	// GET pipelines for repository
	type CFAccount struct {
		ID string `json:"_id"`
	}
	type CFPipeline struct {
		ID      string    `json:"_id"`
		Account CFAccount `json:"account"`
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
		return nil, err
	}

	// scan for pipeline ID
	if pipeline != nil {
		log.WithFields(log.Fields{
			"pipeline":   pipeline.ID,
			"account-id": pipeline.Account.ID,
		}).Debug("found pipeline by id")

		// check account match
		if account != pipeline.Account.ID {
			return nil, ErrPipelineNoMatch
		}
		// return pipeline
		return &Pipeline{ID: pipeline.ID, Account: pipeline.Account.ID}, nil
	}

	log.WithField("pipeline", id).Error("failed to find pipeline with id")
	return nil, ErrPipelineNotFound
}

// run Codefresh pipeline
func (api *APIEndpoint) runPipeline(id string, vars map[string]string) (string, error) {
	log.WithField("pipeline", id).Debug("Going to run pipeline")
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
			"pipeline": id,
			"error":    err,
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
		log.Error("close response body error")
		return "", err
	}
	runID := string(respData)

	if resp.StatusCode == http.StatusOK {
		log.WithFields(log.Fields{
			"pipeline": id,
			"run-id":   runID,
		}).Debug("pipeline is running")
	}
	return runID, nil
}

// GetPipeline get existing pipeline
func (api *APIEndpoint) GetPipeline(account, pipelineUID string) (*Pipeline, error) {
	// invoke pipeline by id
	return api.getPipeline(account, pipelineUID)
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
