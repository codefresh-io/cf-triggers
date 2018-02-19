package provider

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// EventProviderService Codefresh Service
	EventProviderService interface {
		GetEventInfo(event, secret string) (*model.EventInfo, error)
		SubscribeToEvent(event, secret string, credentials map[string]string) (*model.EventInfo, error)
		UnsubscribeFromEvent(event string, credentials map[string]string) error
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
	log.WithField("url", url).Debug("Initializing event-provider api")
	endpoint := sling.New().Base(url)
	return &APIEndpoint{endpoint}
}

// GetEventInfo get EventInfo from Event Provider passing event URI
func (api *APIEndpoint) GetEventInfo(event string, secret string) (*model.EventInfo, error) {
	var info model.EventInfo
	path := fmt.Sprint("/event/", url.QueryEscape(event), "/", url.QueryEscape(secret))
	log.WithField("path", path).Debug("GET event info from event provider")
	resp, err := api.endpoint.New().Get(path).ReceiveSuccess(&info)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("event-provider api error %s", http.StatusText(resp.StatusCode))
	}

	return &info, err
}

// SubscribeToEvent configure remote system through event provider to subscribe for desired event
func (api *APIEndpoint) SubscribeToEvent(event, secret string, credentials map[string]string) (*model.EventInfo, error) {
	var info model.EventInfo
	// encode credentials to pass them in url
	creds, _ := json.Marshal(credentials)
	encoded := base64.StdEncoding.EncodeToString(creds)
	// invoke POST method passing credentials as base64 encoded string; receive eventinfo on success
	path := fmt.Sprint("/event/", url.QueryEscape(event), "/", url.QueryEscape(secret), "/", url.QueryEscape(encoded))
	log.WithField("path", path).Debug("POST event to event provider")
	resp, err := api.endpoint.New().Post(path).ReceiveSuccess(&info)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotImplemented {
		return nil, ErrNotImplemented
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("event-provider api error %s", http.StatusText(resp.StatusCode))
	}

	return &info, err
}

// UnsubscribeFromEvent configure remote system through event provider to unsubscribe for desired event
func (api *APIEndpoint) UnsubscribeFromEvent(event string, credentials map[string]string) error {
	// encode credentials to pass them in url
	creds, _ := json.Marshal(credentials)
	encoded := base64.StdEncoding.EncodeToString(creds)
	// invoke DELETE method passing credentials as base64 encoded string
	path := fmt.Sprint("/event/", url.QueryEscape(event), "/", url.QueryEscape(encoded))
	log.WithField("path", path).Debug("DELETE event from event provider")
	resp, err := api.endpoint.New().Delete(path).Receive(nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotImplemented {
		return ErrNotImplemented
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("event-provider api error %s", http.StatusText(resp.StatusCode))
	}

	return err
}
