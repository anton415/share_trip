package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

type PublishTripCommand struct {
	TripID   uuid.UUID
	ClientID uuid.UUID
}

func (s *TripService) PublishTrip(ctx context.Context, command PublishTripCommand) (uuid.UUID, error) {
	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		s.metrics.TripPublishTotal.WithLabelValues(result).Inc()
		s.metrics.TripPublishDuration.WithLabelValues(result).
			Observe(time.Since(started).Seconds())
	}()

	resp, err := tx(ctx, s.pool, func(tx pgx.Tx) (*domain.PublishTripResponse, error) {
		return s.tripUsecase.PublishTrip(ctx, tx, domain.PublishTripRequest{
			TripID:   command.TripID,
			ClientID: command.ClientID,
		})
	})
	if err != nil {
		result = publishTripResult(err)
		return uuid.Nil, fmt.Errorf("publish trip transaction: %w", err)
	}

	return resp.TripID, nil
}
