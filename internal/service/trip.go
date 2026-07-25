package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"job4j.ru/share-trip/internal/domain"

	"job4j.ru/share-trip/internal/observability/logctx"
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
	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "CreateTrip"),
	)

	logger.Info("create trip started")

	if err := validateCreateTripCommand(command); err != nil {
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
		logger.Error(
			"create trip failed",
			slog.Any("error", err),
		)
		return domain.Trip{}, fmt.Errorf("create trip transaction: %w", err)
	}

	logger.Info(
		"create trip completed",
		slog.String("trip_id", resp.Trip.ID),
	)

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
	TripID   uuid.UUID
	ClientID uuid.UUID
}

func (s *TripService) PublishTrip(ctx context.Context, command PublishTripCommand) (string, error) {
	resp, err := tx(ctx, s.pool, func(tx pgx.Tx) (*domain.PublishTripResponse, error) {
		return s.tripUsecase.PublishTrip(ctx, tx, domain.PublishTripRequest{
			TripID:   command.TripID.String(),
			ClientID: command.ClientID.String(),
		})
	})
	if err != nil {
		return "", fmt.Errorf("publish trip transaction: %w", err)
	}

	return resp.TripID, nil
}
