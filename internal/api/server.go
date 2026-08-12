package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type TripService interface {
	CreateTrip(ctx context.Context, command service.CreateTripCommand) (domain.Trip, error)
	GetTripByID(ctx context.Context, id string) (domain.Trip, error)
	PublishTrip(ctx context.Context, command service.PublishTripCommand) (string, error)
}

type Server struct {
	trips    TripService
	db       Pinger
	registry prometheus.Gatherer
}

func NewServer(
	trips TripService,
	db Pinger,
	registry prometheus.Gatherer,
) *Server {
	return &Server{
		trips:    trips,
		db:       db,
		registry: registry,
	}
}

func (s *Server) Route(router fiber.Router) {
	router.Get("/ready", s.ready)
	router.Get("/metrics", s.metricsHandler())

	trips := router.Group("/trip")
	trips.Post("/create", s.createTrip)
	trips.Post("/publish", s.moveTripDraftToPublished)
	trips.Get("/:id", s.getTripByID)
}
