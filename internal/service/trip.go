package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"job4j.ru/share-trip/internal/domain"
	"strings"
	"time"
)

type TripRepository interface {
	domain.TripRepository

	GetTripByID(ctx context.Context, id string) (domain.Trip, error)
}

type CreateTripCommand struct {
	DriverID      string
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
}

var ErrValidation = errors.New("validation error")
var ErrNotFound = errors.New("not found")

type TripService struct {
	repo        TripRepository
	pool        *pgxpool.Pool
	tripUsecase *domain.TripUsecase
}

func NewTripService(repo TripRepository, pool *pgxpool.Pool) *TripService {
	return &TripService{
		repo:        repo,
		pool:        pool,
		tripUsecase: domain.NewTripUsecase(repo),
	}
}

func (s *TripService) CreateTrip(ctx context.Context, command CreateTripCommand) (domain.Trip, error) {
	if err := validateCreateTripCommand(command); err != nil {
		return domain.Trip{}, err
	}

	resp, err := tx(ctx, s.pool, func(tx pgx.Tx) (*domain.CreateTripResponse, error) {
		return s.tripUsecase.CreateTrip(ctx, tx, domain.CreateTripRequest{
			DriverID:      command.DriverID,
			FromPoint:     command.FromPoint,
			ToPoint:       command.ToPoint,
			DepartureTime: command.DepartureTime,
			Seats:         command.Seats,
		})
	})
	if err != nil {
		return domain.Trip{}, fmt.Errorf("create trip transaction: %w", err)
	}

	return resp.Trip, nil
}

func (s *TripService) GetTripByID(ctx context.Context, id string) (domain.Trip, error) {
	return s.repo.GetTripByID(ctx, id)
}

func validateCreateTripCommand(command CreateTripCommand) error {
	if strings.TrimSpace(command.DriverID) == "" {
		return errors.Join(ErrValidation, errors.New("driver_id is required"))
	}

	if strings.TrimSpace(command.FromPoint) == "" {
		return errors.Join(ErrValidation, errors.New("from_point is required"))
	}

	if strings.TrimSpace(command.ToPoint) == "" {
		return errors.Join(ErrValidation, errors.New("to_point is required"))
	}

	if !command.DepartureTime.After(time.Now()) {
		return errors.Join(ErrValidation, errors.New("departure_time must be in the future"))
	}

	if command.Seats <= 0 {
		return errors.Join(ErrValidation, errors.New("seats must be greater than zero"))
	}

	return nil
}

type PublishTripCommand struct {
	TripID   string
	ClientID string
}

func (s *TripService) PublishTrip(ctx context.Context, command PublishTripCommand) (string, error) {
	if strings.TrimSpace(command.TripID) == "" {
		return "", errors.Join(ErrValidation, errors.New("trip_id is required"))
	}
	if strings.TrimSpace(command.ClientID) == "" {
		return "", errors.Join(ErrValidation, errors.New("client_id is required"))
	}

	resp, err := tx(ctx, s.pool, func(tx pgx.Tx) (*domain.PublishTripResponse, error) {
		return s.tripUsecase.PublishTrip(ctx, tx, domain.PublishTripRequest{
			TripID:   command.TripID,
			ClientID: command.ClientID,
		})
	})
	if err != nil {
		return "", fmt.Errorf("publish trip transaction: %w", err)
	}

	return resp.TripID, nil
}
