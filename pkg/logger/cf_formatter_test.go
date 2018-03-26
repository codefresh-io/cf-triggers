package logger

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus"
)

func TestCFFormatter_Format(t *testing.T) {
	type fields struct {
		TimestampFormat string
	}
	type args struct {
		entry *logrus.Entry
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    _entry
		wantErr bool
	}{
		{
			name: "format full log message",
			args: args{
				entry: &logrus.Entry{
					Level: logrus.DebugLevel,
					Data: logrus.Fields{
						FieldNamespace:     "test-namespace",
						FieldService:       "test-service",
						FieldCorrelationID: "test-correlation-id",
						FieldUserID:        "test-user-id",
						FieldUserName:      "test-user-name",
						"name":             "bob",
						"tag":              "latest",
						"answer":           42,
					},
					Time:    time.Date(2018, time.November, 10, 23, 0, 0, 0, time.UTC),
					Message: "test message",
				},
			},
			want: _entry{
				Metadata: _metadata{
					Namespace:     "test-namespace",
					Service:       "test-service",
					CorrelationID: "test-correlation-id",
					Level:         "debug",
					Time:          "2018-11-10T23:00:00Z",
					User:          _user{ID: "test-user-id", Name: "test-user-name"},
				},
				Data: _data{
					Message: "test message",
					Fields: map[string]interface{}{
						"name":   "bob",
						"tag":    "latest",
						"answer": float64(42),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &CFFormatter{
				TimestampFormat: tt.fields.TimestampFormat,
			}
			got, err := f.Format(tt.args.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("CFFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var entry _entry
			json.Unmarshal(got, &entry)
			assert.EqualValues(t, &tt.want, &entry)
		})
	}
}
