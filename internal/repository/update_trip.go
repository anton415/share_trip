package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func (r *PostgresTripRepository) Update(
	ctx context.Context,
	tx pgx.Tx,
	trip domain.Trip,
) (domain.Trip, error) {
	ctx, span := otel.Tracer("TripRepository").
		Start(ctx, "TripRepository.Update")
	defer span.End()

	span.SetAttributes(
		attribute.String("trip_id", trip.ID.String()),
		attribute.String("trip_status", string(trip.Status)),
	)

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
				id,
				driver_id,
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
			SELECT updated_trip.id, old_trip.status, updated_trip.status::trip_status
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
