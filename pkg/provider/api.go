package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/codefresh-io/hermes/pkg/codefresh"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// EventProviderService Codefresh Service
	EventProviderService interface {
		GetEventInfo(ctx context.Context, event, secret string) (*model.EventInfo, error)
		SubscribeToEvent(ctx context.Context, event, secret string, credentials map[string]string) (*model.EventInfo, error)
		UnsubscribeFromEvent(ctx context.Context, event string, credentials map[string]string) error
	}

	// APIError api error message
	APIError struct {
		Message string `json:"error,omitempty"`
	}

	// APIEndpoint Event Provider API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
	}
)

// ErrNotImplemented error
var ErrNotImplemented = errors.New("method not implemented")

// NewEventProviderEndpoint create new Event Provider API endpoint from url and API token
func NewEventProviderEndpoint(url string) EventProviderService {
	log.WithField("url", url).Debug("initializing event-provider api")
	endpoint := sling.New().Base(url)
	return &APIEndpoint{endpoint}
}

// create new Event Provider API endpoint from url http client (usually mock)
func newTestEventProviderEndpoint(doer sling.Doer, url string) EventProviderService {
	log.WithField("url", url).Debug("initializing event-provider api (test mode)")
	endpoint := sling.New().Doer(doer).Base(url)
	return &APIEndpoint{endpoint}
}

func setContext(ctx context.Context, req *sling.Sling) *sling.Sling {
	// set request ID header
	v := ctx.Value(model.ContextRequestID)
	if val, ok := v.(string); ok {
		req = req.Set(codefresh.RequestID, val)
	}
	// set Authenticated Entry JSON header
	v = ctx.Value(model.ContextAuthEntity)
	if val, ok := v.(string); ok {
		req = req.Set(codefresh.AuthEntity, val)
	}

	return req
}

// GetEventInfo get EventInfo from Event Provider passing event URI
func (api *APIEndpoint) GetEventInfo(ctx context.Context, event string, secret string) (*model.EventInfo, error) {
	var info model.EventInfo
	var apiError APIError
	path := fmt.Sprint("/event/", url.QueryEscape(event), "/", secret)
	log.WithField("path", path).Debug("GET event info from event provider")
	resp, err := setContext(ctx, api.endpoint.New()).Get(path).Receive(&info, &apiError)
	if err != nil && err != io.EOF {
		log.WithError(err).Error("failed to set context for method call")
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		log.WithField("error", apiError.Message).Error("event-provider get info failed")
		return nil, fmt.Errorf("event-provider api error %s, http-code: %d", apiError.Message, resp.StatusCode)
	}

	return &info, err
}

// SubscribeToEvent configure remote system through event provider to subscribe for desired event
func (api *APIEndpoint) SubscribeToEvent(ctx context.Context, event, secret string, credentials map[string]string) (*model.EventInfo, error) {
	var info model.EventInfo
	var apiError APIError
	// encode credentials to pass them in url
	creds, err := json.Marshal(credentials)
	if err != nil {
		log.WithError(err).Error("failed to serialize credentials into JSON")
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(creds)
	// invoke POST method passing credentials as base64 encoded string; receive eventinfo on success
	path := fmt.Sprint("/event/", url.QueryEscape(event), "/", secret, "/", encoded)
	log.WithField("path", path).Debug("POST event to event provider")
	resp, err := setContext(ctx, api.endpoint.New()).Post(path).Receive(&info, &apiError)
	if err != nil && err != io.EOF {
		log.WithError(err).Error("failed to invoke method")
		return nil, err
	}
	if resp.StatusCode == http.StatusNotImplemented {
		log.Warn("method not implemented")
		return nil, ErrNotImplemented
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		log.WithFields(log.Fields{
			"http.status": resp.StatusCode,
			"error":       apiError.Message,
		}).Error("event-provider api method failed")
		return nil, fmt.Errorf("event-provider api error: %s, http-status: %s", apiError.Message, http.StatusText(resp.StatusCode))
	}

	return &info, err
}

// UnsubscribeFromEvent configure remote system through event provider to unsubscribe for desired event
func (api *APIEndpoint) UnsubscribeFromEvent(ctx context.Context, event string, credentials map[string]string) error {
	var apiError APIError
	// encode credentials to pass them in url
	creds, err := json.Marshal(credentials)
	if err != nil {
		log.WithError(err).Error("failed to serialize credentials into JSON")
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(creds)
	// invoke DELETE method passing credentials as base64 encoded string
	path := fmt.Sprint("/event/", url.QueryEscape(event), "/", encoded)
	log.WithField("path", path).Debug("DELETE event from event provider")
	resp, err := setContext(ctx, api.endpoint.New()).Delete(path).Receive(nil, &apiError)
	if err != nil && err != io.EOF {
		log.WithError(err).Error("failed to invoke method")
		return err
	}
	if resp.StatusCode == http.StatusNotImplemented {
		log.Warn("method not implemented")
		return ErrNotImplemented
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		log.WithFields(log.Fields{
			"http.status": resp.StatusCode,
			"error":       apiError.Message,
		}).Error("event-provider api method failed")
		return fmt.Errorf("event-provider api error: %s ,http-status: %s", apiError.Message, http.StatusText(resp.StatusCode))
	}

	return err
}
