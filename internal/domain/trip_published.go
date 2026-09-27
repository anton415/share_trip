package domain

import "time"

type TripPublished struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	TripID     string    `json:"trip_id"`
	DriverID   string    `json:"driver_id"`
	CompanyID  string    `json:"company_id"`
	OccurredAt time.Time `json:"occurred_at"`
}
