package codefresh

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
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
		GetPipeline(ctx context.Context, account, id string) (*Pipeline, error)
		RunPipeline(accountID string, id string, vars map[string]string, event model.NormalizedEvent) (string, error)
		PublishEvent(ctx context.Context, account string, eventURI string, event model.NormalizedEvent) error
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
		log.Error(msg)
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

// publish event to Codefresh Eventbus
func (api *APIEndpoint) publishEvent(ctx context.Context, account string, eventURI string, event model.NormalizedEvent) error {
	/*
		{
		"publisher": "trigger-manager",
		"name": "event.created",
		"aggregateId": "auto-generated ULID",
		"accountId": "account ID",
		"props": {
			"URI": "event-uri",
			"Original": "base64 encoded original event payload"
			"Variables" "hash map of selected event variables"
		}
	*/
	type Payload struct {
		URI       string            `json:"uri"`
		Original  string            `json:"original,omitempty"`
		Variables map[string]string `json:"variables,omitempty"`
	}
	type EventRequest struct {
		Publisher   string  `json:"publisher,omitempty"`
		Name        string  `json:"name,omitempty"`
		AggregateID string  `json:"aggregateId,omitempty"`
		AccountID   string  `json:"accountId,omitempty"`
		Props       Payload `json:"props,omitempty"`
	}
	log.WithField("event", event).Debug("publish event to eventbus")
	// Sling API
	apiClient := api.endpoint.New()
	// set authenticated entity header from context
	if authEntity, ok := ctx.Value(model.ContextAuthEntity).(string); ok {
		apiClient = apiClient.Set(AuthEntity, authEntity)
	}
	// generate ULID for normalized event
	aggregationID, err := util.GenerateULID()
	if err != nil {
		log.WithError(err).Error("failed to generate event ULID")
	}
	// prepare event
	payload := Payload{
		URI:       eventURI,
		Original:  event.Original,
		Variables: event.Variables,
	}
	body := &EventRequest{
		Publisher:   "trigger-manager",
		Name:        "event.created",
		AccountID:   account,
		AggregateID: aggregationID,
		Props:       payload,
	}
	// call codefresh API
	log.Debug("publishing event with cfapi")
	resp, err := apiClient.Post(fmt.Sprint("api/system/publish-event")).BodyJSON(body).ReceiveSuccess(nil)
	err = checkResponse("publish event", err, resp)
	if err != nil {
		log.WithError(err).Error("failed to publish event")
	}
	return err
}

// find Codefresh pipeline by name and repo details (owner and name)
func (api *APIEndpoint) getPipeline(ctx context.Context, account, id string) (*Pipeline, error) {
	log.WithField("pipeline", id).Debug("getting pipeline")
	// GET pipelines for repository
	type CFAccount struct {
		ID string `json:"_id"`
	}
	type CFPipeline struct {
		ID      string    `json:"id"`
		Account CFAccount `json:"account"`
	}
	pipeline := new(CFPipeline)
	var resp *http.Response
	var err error
	// Sling API
	apiClient := api.endpoint.New()
	// set authenticated entity header from context
	if authEntity, ok := ctx.Value(model.ContextAuthEntity).(string); ok {
		apiClient = apiClient.Set(AuthEntity, authEntity)
	}
	// call codefresh API
	if api.internal {
		// use internal cfapi - another endpoint and need to add account
		log.Debug("get pipelines, using internal cfapi")
		resp, err = apiClient.Get(fmt.Sprint("api/pipelines/", account, "/", id)).ReceiveSuccess(pipeline)
	} else {
		// use public cfapi
		log.Debug("get pipelines, using public cfapi")
		resp, err = apiClient.Get(fmt.Sprint("api/pipelines/", id)).ReceiveSuccess(pipeline)
	}
	err = checkResponse("get pipelines", err, resp)
	if err != nil {
		log.WithError(err).Error("failed to get pipelines")
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
			log.Error("pipeline does not match account")
			return nil, ErrPipelineNoMatch
		}
		// return pipeline
		return &Pipeline{ID: pipeline.ID, Account: pipeline.Account.ID}, nil
	}

	log.WithField("pipeline", id).Error("failed to find pipeline with id")
	return nil, ErrPipelineNotFound
}

// run Codefresh pipeline
func (api *APIEndpoint) runPipeline(accountID string, id string, vars map[string]string, event model.NormalizedEvent) (string, error) {
	log.WithField("pipeline", id).Debug("Going to run pipeline")
	type BuildRequest struct {
		Branch    string                `json:"branch,omitempty"`
		Variables map[string]string     `json:"variables,omitempty"`
		Event     model.NormalizedEvent `json:"event,omitempty"`
	}

	// start new run
	body := &BuildRequest{
		Branch:    "master",
		Variables: preprocessVariables(vars),
		Event:     event,
	}
	req, err := api.endpoint.New().Post(fmt.Sprint("api/builds/", accountID, "/", id)).BodyJSON(body).Request()
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
		log.WithError(err).Error("failed to run pipeline")
		return "", err
	}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("close response body error")
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
func (api *APIEndpoint) GetPipeline(ctx context.Context, account, pipelineUID string) (*Pipeline, error) {
	// invoke pipeline by id
	return api.getPipeline(ctx, account, pipelineUID)
}

// RunPipeline run Codefresh pipeline
func (api *APIEndpoint) RunPipeline(accountID string, pipelineUID string, vars map[string]string, event model.NormalizedEvent) (string, error) {
	// invoke pipeline by id
	return api.runPipeline(accountID, pipelineUID, vars, event)
}

// PublishEvent publish trigger-event normalized event to eventbus
func (api *APIEndpoint) PublishEvent(ctx context.Context, account string, eventURI string, event model.NormalizedEvent) error {
	return api.publishEvent(ctx, account, eventURI, event)
}

// Ping Codefresh API
func (api *APIEndpoint) Ping() error {
	return api.ping()
}
