package provider

import (
	"fmt"
	"net/http"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// EventProviderService Codefresh Service
	EventProviderService interface {
		GetEvent(event string, secret string) (*model.EventInfo, error)
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

// GetEvent get EventInfo from Event Provider passing event URI
func (api *APIEndpoint) GetEvent(event string, secret string) (*model.EventInfo, error) {
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
