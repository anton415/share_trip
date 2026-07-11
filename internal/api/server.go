package api

import (
	"context"
	"log/slog"
	"time"

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
	router.Post("/trip/create", s.createTrip)
	router.Post("/trip/publish", s.publishTrip)
	router.Get("/trip/:id", s.getTripByID)
}

func (s *Server) ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		slog.Error("database readiness check failed", "error", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(errorResponse{
			Code:    "SERVICE_UNAVAILABLE",
			Message: "database unavailable",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}
