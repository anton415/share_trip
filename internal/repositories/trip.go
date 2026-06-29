package repositories

import (
	"context"
	"errors"
	"fmt"

	"job4j.ru/share-trip/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTripRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTripRepository(pool *pgxpool.Pool) *PostgresTripRepository {
	return &PostgresTripRepository{pool: pool}
}

func (r *PostgresTripRepository) CreateTrip(ctx context.Context, command service.CreateTripCommand) (service.Trip, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.Trip{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var trip service.Trip
	var status string

	err = tx.QueryRow(ctx, `
		INSERT INTO trips (
			driver_id,
			from_point,
			to_point,
			departure_time,
			available_seats
		)
		VALUES ($1, $2, $3, $4, $5)
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
		command.DriverID,
		command.FromPoint,
		command.ToPoint,
		command.DepartureTime,
		command.Seats,
	).Scan(
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
	if err != nil {
		return service.Trip{}, fmt.Errorf("insert trip: %w", err)
	}

	trip.Status = service.TripStatus(status)

	_, err = tx.Exec(ctx, `
		INSERT INTO trip_history (trip_id, from_status, to_status)
		VALUES ($1, NULL, $2)
	`, trip.ID, service.TripStatusDraft)
	if err != nil {
		return service.Trip{}, fmt.Errorf("insert trip history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return service.Trip{}, fmt.Errorf("commit transaction: %w", err)
	}

	return trip, nil
}

func (r *PostgresTripRepository) GetTripByID(ctx context.Context, id string) (service.Trip, error) {
	var trip service.Trip
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
		return service.Trip{}, service.ErrNotFound
	}
	if err != nil {
		return service.Trip{}, fmt.Errorf("get trip by id: %w", err)
	}

	trip.Status = service.TripStatus(status)

	return trip, nil
}
