package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

type TripRepository interface {
	CreateTrip(ctx context.Context, command CreateTripCommand) (Trip, error)
	GetTripByID(ctx context.Context, id string) (Trip, error)
}

type TripStatus string

const (
	TripStatusDraft TripStatus = "draft"
)

type Trip struct {
	ID            string
	DriverID      string
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        TripStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
	repo TripRepository
}

func NewTripService(repo TripRepository) *TripService {
	return &TripService{repo: repo}
}

func (s *TripService) CreateTrip(ctx context.Context, command CreateTripCommand) (Trip, error) {
	if err := validateCreateTripCommand(command); err != nil {
		return Trip{}, err
	}

	return s.repo.CreateTrip(ctx, command)
}

func (s *TripService) GetTripByID(ctx context.Context, id string) (Trip, error) {
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
