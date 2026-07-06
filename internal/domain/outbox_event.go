package domain

import "encoding/json"

type OutboxEvent struct {
	EventName   string          `json:"event_name"`
	AggregateID string          `json:"aggregate_id"`
	Payload     json.RawMessage `json:"payload"`
}
