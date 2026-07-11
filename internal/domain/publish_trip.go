package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type PublishTripRequest struct {
	TripID   string
	ClientID string
}

type PublishTripResponse struct {
	TripID string
}

func (u *TripUsecase) PublishTrip(
	ctx context.Context,
	tx pgx.Tx,
	req PublishTripRequest,
) (*PublishTripResponse, error) {
	trip, err := u.tripRepo.GetForUpdateByID(ctx, tx, req.TripID)
	if err != nil {
		return nil, fmt.Errorf("tripRepo.GetForUpdateByID: %w", err)
	}

	if trip.DriverID != req.ClientID {
		return nil, fmt.Errorf(
			"%w: client %s is not driver of trip %s",
			ErrForbidden,
			req.ClientID,
			req.TripID,
		)
	}

	if trip.Status == TripStatusPublished {
		return nil, fmt.Errorf(
			"%w: trip %s",
			ErrTripAlreadyPublished,
			trip.ID,
		)
	}

	if trip.Status != TripStatusDraft {
		return nil, fmt.Errorf(
			"%w: invalid trip status: expected %s, got %s",
			ErrConflict,
			TripStatusDraft,
			trip.Status,
		)
	}

	trip.Status = TripStatusPublished

	updatedTrip, err := u.tripRepo.Update(ctx, tx, trip)
	if err != nil {
		return nil, fmt.Errorf("tripRepo.Update: %w", err)
	}

	payload, err := json.Marshal(struct {
		TripID string `json:"trip_id"`
	}{
		TripID: updatedTrip.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal trip published payload: %w", err)
	}

	err = u.tripRepo.CreateOutboxEvent(ctx, tx, OutboxEvent{
		EventName:   "trip_published",
		AggregateID: updatedTrip.ID,
		Payload:     payload,
	})
	if err != nil {
		return nil, fmt.Errorf("tripRepo.CreateOutboxEvent: %w", err)
	}

	return &PublishTripResponse{TripID: updatedTrip.ID}, nil
}
