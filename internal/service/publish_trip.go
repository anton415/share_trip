package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/events"
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

	ctx, span := otel.Tracer("TripService").
		Start(ctx, "TripService.PublishTrip")
	defer span.End()

	span.SetAttributes(
		attribute.String("trip_id", command.TripID.String()),
		attribute.String("client_id", command.ClientID.String()),
	)

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

	err = s.publisher.PublishTripPublished(ctx, events.TripPublished{
		EventID:    uuid.NewString(),
		EventType:  "TripPublished",
		TripID:     resp.TripID.String(),
		DriverID:   command.ClientID.String(),
		CompanyID:  command.ClientID.String(),
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		result = observability.ResultInternalError
		return uuid.Nil, fmt.Errorf("publish TripPublished event: %w", err)
	}

	return resp.TripID, nil
}
