package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func (r *PostgresTripRepository) GetForUpdateByID(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) (domain.Trip, error) {
	ctx, span := otel.Tracer("TripRepository").
		Start(ctx, "TripRepository.GetForUpdateByID")
	defer span.End()

	span.SetAttributes(attribute.String("trip_id", id.String()))

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
			id,
			driver_id,
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
	span.SetAttributes(attribute.String("trip_status", string(trip.Status)))

	return trip, nil
}
