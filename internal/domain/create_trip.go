package domain

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

type CreateTripRequest struct {
	DriverID      string
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
}

type CreateTripResponse struct {
	Trip Trip
}

type TripRepository interface {
	Create(ctx context.Context, tx pgx.Tx, trip Trip) (Trip, error)
	GetForUpdateByID(ctx context.Context, tx pgx.Tx, id string) (Trip, error)
	Update(ctx context.Context, tx pgx.Tx, trip Trip) (Trip, error)
	CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event OutboxEvent) error
}

type TripUsecase struct {
	tripRepo TripRepository
}

func NewTripUsecase(tripRepo TripRepository) *TripUsecase {
	return &TripUsecase{tripRepo: tripRepo}
}

func (u *TripUsecase) CreateTrip(
	ctx context.Context,
	tx pgx.Tx,
	req CreateTripRequest,
) (*CreateTripResponse, error) {
	trip, err := u.tripRepo.Create(ctx, tx, Trip{
		DriverID:      req.DriverID,
		FromPoint:     req.FromPoint,
		ToPoint:       req.ToPoint,
		DepartureTime: req.DepartureTime,
		Seats:         req.Seats,
		Status:        TripStatusDraft,
	})
	if err != nil {
		return nil, fmt.Errorf("tripRepo.Create: %w", err)
	}

	return &CreateTripResponse{
		Trip: trip,
	}, nil
}
