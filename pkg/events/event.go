package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Event         string      `json:"event"`
	Version       string      `json:"version"`
	Timestamp     time.Time   `json:"timestamp"`
	Payload       interface{} `json:"payload"`
	TraceID       string      `json:"traceId"`
	CorrelationID string      `json:"correlationId"`
}

type Headers struct {
	TraceID       string
	CorrelationID string
	Service       string
}

func NewEvent(eventName, version string, payload interface{}, headers Headers) *Event {
	return &Event{
		Event:         eventName,
		Version:       version,
		Timestamp:     time.Now().UTC(),
		Payload:       payload,
		TraceID:       headers.TraceID,
		CorrelationID: headers.CorrelationID,
	}
}

func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Event) GetRoutingKey() string {
	return e.Event + "." + e.Version
}

func GenerateTraceID() string {
	return uuid.New().String()
}

func GenerateCorrelationID() string {
	return uuid.New().String()
}
