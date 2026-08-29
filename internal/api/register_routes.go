package api

import (
	"github.com/gofiber/fiber/v2"

	"job4j.ru/share-trip/internal/middleware"
)

func (s *Server) RegisterRoutes(
	router fiber.Router,
	keycloakAuth fiber.Handler,
	keycloakClientID string,
) {
	router.Get("/ready", s.ready)
	router.Get("/metrics", s.metricsHandler())

	trips := router.Group("/trip", keycloakAuth)
	trips.Post(
		"/create",
		middleware.RequireClientRole(keycloakClientID, "client"),
		s.createTrip,
	)
	trips.Post("/publish", s.moveTripDraftToPublished)
	trips.Get(
		"/:id",
		middleware.RequireClientRole(keycloakClientID, "client"),
		s.getTripByID,
	)
}
