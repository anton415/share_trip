package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func (r *PostgresTripRepository) GetTripByID(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
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
