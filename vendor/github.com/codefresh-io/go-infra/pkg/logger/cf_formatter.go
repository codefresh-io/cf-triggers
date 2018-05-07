package logger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// default time format
const defaultTimestampFormat = time.RFC3339

// Default key names for the metadata fields
const (
	FieldCorrelationID = "correlationID"
	FieldNamespace     = "namespace"
	FieldService       = "service"
	FieldAuthName      = "authName"
	FieldAuthID        = "authID"
	FieldAuthType      = "authType"
	FieldNewRelicTxn   = "newRelicTransaction"
)

type _authEntity struct {
	ID   string `json:"id"`
	Name string `json:"user"`
	Type string `json:"type"`
}

type _metadata struct {
	Namespace     string      `json:"namespace,omitempty"`
	Service       string      `json:"service,omitempty"`
	Time          string      `json:"time"`
	Level         string      `json:"level"`
	CorrelationID string      `json:"correlationId,omitempty"`
	AuthEntity    _authEntity `json:"authenticatedEntity,omitempty"`
}

type _data struct {
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields"`
}

type _entry struct {
	Metadata _metadata `json:"metadata"`
	Data     _data     `json:"data"`
}

// CFFormatter formats logs into parsable json
type CFFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string
}

// Format renders a single log entry
func (f *CFFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// new log entry
	logEntry := new(_entry)
	logEntry.Data.Fields = make(map[string]interface{})

	// fill fields
	for k, v := range entry.Data {
		// skip some fields, use them for metadata later
		switch k {
		case
			FieldCorrelationID,
			FieldNamespace,
			FieldService,
			FieldAuthName,
			FieldAuthID,
			FieldAuthType,
			FieldNewRelicTxn:
			continue
		}
		// get value, for error use Error()
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			logEntry.Data.Fields[k] = v.Error()
		default:
			logEntry.Data.Fields[k] = v
		}
	}

	// override default time format
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	// set message
	logEntry.Data.Message = entry.Message

	// fill metadata
	logEntry.Metadata.Level = entry.Level.String()
	logEntry.Metadata.Time = entry.Time.Format(timestampFormat)
	logEntry.Metadata.CorrelationID = getFieldByName(FieldCorrelationID, entry.Data)
	logEntry.Metadata.Namespace = getFieldByName(FieldNamespace, entry.Data)
	logEntry.Metadata.Service = getFieldByName(FieldService, entry.Data)
	logEntry.Metadata.AuthEntity = _authEntity{
		ID:   getFieldByName(FieldAuthID, entry.Data),
		Name: getFieldByName(FieldAuthName, entry.Data),
		Type: getFieldByName(FieldAuthType, entry.Data),
	}

	serialized, err := json.Marshal(logEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}

func getFieldByName(name string, fields logrus.Fields) string {
	if value, ok := fields[name]; ok {
		return value.(string)
	}
	return ""
}
