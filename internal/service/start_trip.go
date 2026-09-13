package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
)

func (s *TripService) StartTrip(ctx context.Context, tripID, clientID uuid.UUID) (uuid.UUID, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return uuid.Nil, err
	}
	if trip.DriverID != clientID {
		return uuid.Nil, domain.ErrForbidden
	}
	if trip.Status != domain.TripStatusPublished {
		return uuid.Nil, domain.ErrConflict
	}
	started := time.Now()
	result, err := s.contracts.CheckService(ctx, trip.DriverID.String(), "trip_start")
	logctx.Logger(ctx).Info("contract service check", "trip_id", tripID, "company_id", trip.DriverID,
		"service_code", "trip_start", "allowed", result.Allowed, "reason", result.Reason,
		"duration", time.Since(started), "error", err)
	if err != nil {
		return uuid.Nil, err
	}
	if !result.Allowed {
		return uuid.Nil, domain.ErrForbidden
	}
	updated, err := tx(ctx, s.pool, func(dbtx pgx.Tx) (*domain.Trip, error) {
		trip, err := s.repo.GetForUpdateByID(ctx, dbtx, tripID)
		if err != nil {
			return nil, err
		}
		if trip.DriverID != clientID {
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
