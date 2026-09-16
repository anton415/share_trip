package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
)

type StartTripCommand struct {
	TripID   uuid.UUID
	ClientID uuid.UUID
}

func (s *TripService) StartTrip(ctx context.Context, command StartTripCommand) (uuid.UUID, error) {
	started := time.Now()
	result, err := s.contracts.CheckService(ctx, command.ClientID.String(), "trip_start")
	logctx.Logger(ctx).Info("contract service check", "trip_id", command.TripID, "company_id", command.ClientID,
		"service_code", "trip_start", "allowed", result.Allowed, "reason", result.Reason,
		"duration", time.Since(started), "error", err)
	if err != nil {
		return uuid.Nil, err
	}
	if !result.Allowed {
		return uuid.Nil, domain.ErrForbidden
	}
	updated, err := tx(ctx, s.pool, func(dbtx pgx.Tx) (*domain.Trip, error) {
		trip, err := s.repo.GetForUpdateByID(ctx, dbtx, command.TripID)
		if err != nil {
			return nil, err
		}
		if trip.DriverID != command.ClientID {
			return nil, domain.ErrForbidden
		}
		if err := trip.Start(); err != nil {
			return nil, err
		}
		trip, err = s.repo.Update(ctx, dbtx, trip)
		return &trip, err
	})
	if err != nil {
		return uuid.Nil, err
	}
	return updated.ID, nil
}
