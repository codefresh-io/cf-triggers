package logger

import (
	"errors"

	"github.com/newrelic/go-agent"
	"github.com/sirupsen/logrus"
)

type NewRelicLogrusHook struct {
	Application newrelic.Application
	LogLevels   []logrus.Level
}

// NewNewRelicLogrusHook create NewRelic Logrus hook
func NewNewRelicLogrusHook(app newrelic.Application, levels []logrus.Level) *NewRelicLogrusHook {
	return &NewRelicLogrusHook{
		Application: app,
		LogLevels:   levels,
	}
}

// Levels return levels for hook
func (n *NewRelicLogrusHook) Levels() []logrus.Level {
	return n.LogLevels
}

// Fire fire logrus event hook
func (n *NewRelicLogrusHook) Fire(entry *logrus.Entry) error {
	// Hacky. We don't know what transaction we're in so we
	// just start a new one specific to error reporting.
	txn := n.Application.StartTransaction("errorTxn", nil, nil)
	for k, v := range entry.Data {
		txn.AddAttribute(k, v)
	}
	txn.NoticeError(errors.New(entry.Message))
	txn.End()

	return nil
}
