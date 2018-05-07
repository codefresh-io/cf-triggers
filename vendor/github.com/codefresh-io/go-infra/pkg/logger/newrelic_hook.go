package logger

import (
	"errors"

	"github.com/newrelic/go-agent"
	"github.com/sirupsen/logrus"
)

// NewRelicLogrusHook logrus hook for new relic
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
	// try to get transaction from fields
	// create new if not found
	var ok bool
	var txn newrelic.Transaction
	if v, exists := entry.Data[FieldNewRelicTxn]; exists {
		if txn, ok = v.(newrelic.Transaction); !ok {
			txn = n.Application.StartTransaction("errorTxn", nil, nil)
		}
	}
	// get other fields
	for k, v := range entry.Data {
		// skip NewRelic field
		if k == FieldNewRelicTxn {
			continue
		}
		// add field as attribute to newrelic transaction
		txn.AddAttribute(k, v)
	}
	txn.NoticeError(errors.New(entry.Message))
	txn.End()

	return nil
}
