package domain

import "time"

const EventTypeTripPublished = "TripPublished"

type TripPublished struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	TraceParent   string    `json:"traceparent"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	CompanyID     string    `json:"company_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}
