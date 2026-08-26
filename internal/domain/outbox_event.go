package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	EventName   string          `json:"event_name"`
	AggregateID uuid.UUID       `json:"aggregate_id"`
	Payload     json.RawMessage `json:"payload"`
}
