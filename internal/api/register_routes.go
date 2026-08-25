package api

import "github.com/gofiber/fiber/v2"

func (s *Server) RegisterRoutes(router fiber.Router) {
	router.Get("/ready", s.ready)
	router.Get("/metrics", s.metricsHandler())

	trips := router.Group("/trip")
	trips.Post("/create", s.createTrip)
	trips.Post("/publish", s.moveTripDraftToPublished)
	trips.Get("/:id", s.getTripByID)
}
