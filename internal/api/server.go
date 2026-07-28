package api

import (
	"context"

	"github.com/gofiber/fiber/v2"

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
	trips TripService
	db    Pinger
}

func NewServer(trips TripService, db Pinger) *Server {
	return &Server{trips: trips, db: db}
}

func (s *Server) Route(router fiber.Router) {
	router.Get("/ready", s.ready)

	trips := router.Group("/trip")
	trips.Post("/create", s.createTrip)
	trips.Post("/publish", s.publishTrip)
	trips.Get("/:id", s.getTripByID)
}
