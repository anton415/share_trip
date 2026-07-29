package repositories

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

type PostgresTripRepository struct {
	pool    *pgxpool.Pool
	metrics *observability.Metrics
}

func NewPostgresTripRepository(
	pool *pgxpool.Pool,
	metrics *observability.Metrics,
) *PostgresTripRepository {
	return &PostgresTripRepository{
		pool:    pool,
		metrics: metrics,
	}
}

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
			id::text,
			driver_id::text,
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
		slog.String("trip_id", created.ID),
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

func (r *PostgresTripRepository) GetForUpdateByID(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (domain.Trip, error) {
	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		r.observeQuery(
			observability.RepositoryOperationTripGetForUpdateByID,
			result,
			started,
		)
	}()

	var trip domain.Trip
	var status string

	err := tx.QueryRow(ctx, `
		SELECT
			id::text,
			driver_id::text,
			from_point,
			to_point,
			departure_time,
			available_seats,
			status::text,
			created_at,
			updated_at
		FROM trips
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(
		&trip.ID,
		&trip.DriverID,
		&trip.FromPoint,
		&trip.ToPoint,
		&trip.DepartureTime,
		&trip.Seats,
		&status,
		&trip.CreatedAt,
		&trip.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		result = observability.ResultNotFound
		return domain.Trip{}, domain.ErrTripNotFound
	}
	if err != nil {
		result = observability.ResultInternalError
		return domain.Trip{}, fmt.Errorf("get trip by id for update: %w", err)
	}

	trip.Status = domain.TripStatus(status)

	return trip, nil
}

func (r *PostgresTripRepository) Update(
	ctx context.Context,
	tx pgx.Tx,
	trip domain.Trip,
) (domain.Trip, error) {
	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		r.observeQuery(
			observability.RepositoryOperationTripUpdate,
			result,
			started,
		)
	}()

	var updated domain.Trip
	var status string

	err := tx.QueryRow(ctx, `
		WITH old_trip AS (
			SELECT status
			FROM trips
			WHERE id = $1
		),
		updated_trip AS (
			UPDATE trips
			SET
				driver_id = $2,
				from_point = $3,
				to_point = $4,
				departure_time = $5,
				available_seats = $6,
				status = $7::trip_status,
				updated_at = NOW()
			WHERE id = $1
			RETURNING
				id::text,
				driver_id::text,
				from_point,
				to_point,
				departure_time,
				available_seats,
				status::text,
				created_at,
				updated_at
		),
		history AS (
			INSERT INTO trip_history (trip_id, from_status, to_status)
			SELECT updated_trip.id::uuid, old_trip.status, updated_trip.status::trip_status
			FROM updated_trip, old_trip
			WHERE old_trip.status <> updated_trip.status::trip_status
		)
		SELECT
			id,
			driver_id,
			from_point,
			to_point,
			departure_time,
			available_seats,
			status,
			created_at,
			updated_at
		FROM updated_trip
	`,
		trip.ID,
		trip.DriverID,
		trip.FromPoint,
		trip.ToPoint,
		trip.DepartureTime,
		trip.Seats,
		trip.Status,
	).Scan(
		&updated.ID,
		&updated.DriverID,
		&updated.FromPoint,
		&updated.ToPoint,
		&updated.DepartureTime,
		&updated.Seats,
		&status,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		result = observability.ResultNotFound
		return domain.Trip{}, domain.ErrTripNotFound
	}
	if err != nil {
		result = observability.ResultInternalError
		return domain.Trip{}, fmt.Errorf("update trip: %w", err)
	}

	updated.Status = domain.TripStatus(status)

	return updated, nil
}

func (r *PostgresTripRepository) CreateOutboxEvent(
	ctx context.Context,
	tx pgx.Tx,
	event domain.OutboxEvent,
) error {
	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		r.observeQuery(
			observability.RepositoryOperationOutboxEventCreate,
			result,
			started,
		)
	}()

	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_event (
			event_name,
			aggregate_id,
			payload
		)
		VALUES ($1, $2, $3)
	`,
		event.EventName,
		event.AggregateID,
		event.Payload,
	)
	if err != nil {
		result = observability.ResultInternalError
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}

func (r *PostgresTripRepository) GetTripByID(ctx context.Context, id string) (domain.Trip, error) {
	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		r.observeQuery(
			observability.RepositoryOperationTripGetByID,
			result,
			started,
		)
	}()

	var trip domain.Trip
	var status string

	err := r.pool.QueryRow(ctx, `
		SELECT
			id::text,
			driver_id::text,
			from_point,
			to_point,
			departure_time,
			available_seats,
			status::text,
			created_at,
			updated_at
		FROM trips
		WHERE id = $1
	`, id).Scan(
		&trip.ID,
		&trip.DriverID,
		&trip.FromPoint,
		&trip.ToPoint,
		&trip.DepartureTime,
		&trip.Seats,
		&status,
		&trip.CreatedAt,
		&trip.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		result = observability.ResultNotFound
		return domain.Trip{}, domain.ErrTripNotFound
	}
	if err != nil {
		result = observability.ResultInternalError
		return domain.Trip{}, fmt.Errorf("get trip by id: %w", err)
	}

	trip.Status = domain.TripStatus(status)

	return trip, nil
}

func (r *PostgresTripRepository) observeQuery(
	operation string,
	result string,
	started time.Time,
) {
	r.metrics.RepositoryQueryTotal.
		WithLabelValues(operation, result).
		Inc()

	r.metrics.RepositoryQueryDuration.
		WithLabelValues(operation, result).
		Observe(time.Since(started).Seconds())
}
