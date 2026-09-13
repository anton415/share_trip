package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type TripService interface {
	CreateTrip(ctx context.Context, command service.CreateTripCommand) (domain.Trip, error)
	GetTripByID(ctx context.Context, id uuid.UUID) (domain.Trip, error)
	PublishTrip(ctx context.Context, command service.PublishTripCommand) (uuid.UUID, error)
	StartTrip(ctx context.Context, tripID, clientID uuid.UUID) (uuid.UUID, error)
}

type Server struct {
	trips    TripService
	db       Pinger
	registry prometheus.Gatherer
}
