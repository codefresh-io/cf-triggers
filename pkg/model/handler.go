package model

import (
	"fmt"
	"net/http"

	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// EventHandlerService Codefresh Service
	EventHandlerService interface {
		GetEventInfo(eventURI string, secret string) (*EventInfo, error)
	}

	// APIEndpoint EventHandler API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
	}
)

// NewEventHandlerEndpoint create new Event Handler API endpoint from url and API token
func NewEventHandlerEndpoint(url string) EventHandlerService {
	log.Debugf("initializing event-handler api %s ...", url)
	endpoint := sling.New().Base(url)
	return &APIEndpoint{endpoint}
}

// GetEventInfo get EventInfo from Event Handler passing eventURI
func (api *APIEndpoint) GetEventInfo(eventURI string, secret string) (*EventInfo, error) {
	var info EventInfo
	resp, err := api.endpoint.New().Get(fmt.Sprint("/event-info/", eventURI, "/", secret)).ReceiveSuccess(&info)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("event-handler api error %s", http.StatusText(resp.StatusCode))
	}

	return &info, err
}
