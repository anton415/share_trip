package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

type CreateTripCommand struct {
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
}

func (s *TripService) CreateTrip(ctx context.Context, command CreateTripCommand) (domain.Trip, error) {
	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		s.metrics.TripCreateTotal.WithLabelValues(result).Inc()
		s.metrics.TripCreateDuration.WithLabelValues(result).
			Observe(time.Since(started).Seconds())
	}()

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "CreateTrip"),
	)

	logger.Info("create trip started")

	if err := validateCreateTripCommand(command); err != nil {
		result = observability.ResultValidationError
		logger.Warn(
			"create trip failed",
			slog.Any("error", err),
		)
		return domain.Trip{}, err
	}

	resp, err := tx(ctx, s.pool, func(tx pgx.Tx) (*domain.CreateTripResponse, error) {
		usecaseResp, err := s.tripUsecase.CreateTrip(ctx, tx, domain.CreateTripRequest{
			DriverID:      command.DriverID,
			FromPoint:     command.FromPoint,
			ToPoint:       command.ToPoint,
			DepartureTime: command.DepartureTime,
			Seats:         command.Seats,
		})
		if err != nil {
			logger.Error(
				"create trip usecase failed",
				slog.Any("error", err),
			)
			return nil, err
		}

		return usecaseResp, nil
	})
	if err != nil {
		result = observability.ResultInternalError
		logger.Error(
			"create trip failed",
			slog.Any("error", err),
		)
		return domain.Trip{}, fmt.Errorf("create trip transaction: %w", err)
	}

	logger.Info(
		"create trip completed",
		slog.String("trip_id", resp.Trip.ID.String()),
	)

	return resp.Trip, nil
}
