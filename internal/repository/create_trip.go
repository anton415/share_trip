package repo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func (r *PostgresTripRepository) Create(
	ctx context.Context,
	tx pgx.Tx,
	trip domain.Trip,
) (domain.Trip, error) {
	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		r.observeQuery(
			observability.RepositoryOperationTripCreate,
			result,
			started,
		)
	}()

	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "TripRepository"),
		slog.String("operation", "Create"),
	)

	logger.Info("insert trip started")

	var created domain.Trip
	var status string

	err := tx.QueryRow(ctx, `
		INSERT INTO trips (
			driver_id,
			from_point,
			to_point,
			departure_time,
			available_seats,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			driver_id,
			from_point,
			to_point,
			departure_time,
			available_seats,
			status::text,
			created_at,
			updated_at
	`,
		trip.DriverID,
		trip.FromPoint,
		trip.ToPoint,
		trip.DepartureTime,
		trip.Seats,
		trip.Status,
	).Scan(
		&created.ID,
		&created.DriverID,
		&created.FromPoint,
		&created.ToPoint,
		&created.DepartureTime,
		&created.Seats,
		&status,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		result = observability.ResultInternalError
		logger.Error(
			"insert trip failed",
			slog.Any("error", err),
		)
		return domain.Trip{}, fmt.Errorf("insert trip: %w", err)
	}

	created.Status = domain.TripStatus(status)

	logger = logger.With(
		slog.String("trip_id", created.ID.String()),
	)

	_, err = tx.Exec(ctx, `
		INSERT INTO trip_history (trip_id, from_status, to_status)
		VALUES ($1, NULL, $2)
	`, created.ID, created.Status)
	if err != nil {
		result = observability.ResultInternalError
		logger.Error(
			"insert trip history failed",
			slog.Any("error", err),
		)
		return domain.Trip{}, fmt.Errorf("insert trip history: %w", err)
	}

	logger.Info("insert trip completed")
	return created, nil
}
