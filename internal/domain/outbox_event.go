package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID            uuid.UUID       `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"`
	Attempts      int             `json:"attempts"`
	LastError     *string         `json:"last_error"`
	CreatedAt     time.Time       `json:"created_at"`
	SentAt        *time.Time      `json:"sent_at"`
}
