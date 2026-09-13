package domain

import (
	"time"

	"github.com/google/uuid"
)

type TripStatus string

const (
	TripStatusDraft     TripStatus = "draft"
	TripStatusPublished TripStatus = "published"
	TripStatusStarted   TripStatus = "started"
)

type Trip struct {
	ID            uuid.UUID
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        TripStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (t *Trip) Start() error {
	if t.Status != TripStatusPublished {
		return ErrConflict
	}
	t.Status = TripStatusStarted
	return nil
}
