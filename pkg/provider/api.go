package provider

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// EventProviderService Codefresh Service
	EventProviderService interface {
		GetEventInfo(event, secret string) (*model.EventInfo, error)
		SubscribeToEvent(event, secret, credentials string) (*model.EventInfo, error)
		UnsubscribeFromEvent(event, credentials string) error
	}

	// APIEndpoint Event Provider API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
	}
)

// NewEventProviderEndpoint create new Event Provider API endpoint from url and API token
func NewEventProviderEndpoint(url string) EventProviderService {
	log.WithField("url", url).Debug("Initializing event-provider api")
	endpoint := sling.New().Base(url)
	return &APIEndpoint{endpoint}
}

// GetEventInfo get EventInfo from Event Provider passing event URI
func (api *APIEndpoint) GetEventInfo(event string, secret string) (*model.EventInfo, error) {
	var info model.EventInfo
	resp, err := api.endpoint.New().Get(fmt.Sprint("/event/", event, "/", secret)).ReceiveSuccess(&info)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("event-provider api error %s", http.StatusText(resp.StatusCode))
	}

	return &info, err
}

// SubscribeToEvent configure remote system through event provider to subscribe for desired event
func (api *APIEndpoint) SubscribeToEvent(event, secret, credentials string) (*model.EventInfo, error) {
	var info model.EventInfo
	// encode credentials to pass them in url
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	// invoke POST method passing credentials as base64 encoded string; receive eventinfo on success
	resp, err := api.endpoint.New().Post(fmt.Sprint("/event/", event, "/", secret, "/", encoded)).ReceiveSuccess(&info)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("event-provider api error %s", http.StatusText(resp.StatusCode))
	}

	return &info, err
}

// UnsubscribeFromEvent configure remote system through event provider to unsubscribe for desired event
func (api *APIEndpoint) UnsubscribeFromEvent(event, credentials string) error {
	// encode credentials to pass them in url
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	// invoke DELETE method passing credentials as base64 encoded string
	resp, err := api.endpoint.New().Delete(fmt.Sprint("/event/", event, "/", encoded)).Receive(nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("event-provider api error %s", http.StatusText(resp.StatusCode))
	}

	return err
}
