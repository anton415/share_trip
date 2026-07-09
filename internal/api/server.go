package api

import "github.com/gofiber/fiber/v2"

type Server struct {
	trips TripService
}

func NewServer(trips TripService) *Server {
	return &Server{trips: trips}
}

func (s *Server) Route(router fiber.Router) {
	router.Post("/trip/create", s.createTrip)
	router.Post("/trip/publish", s.publishTrip)
	router.Get("/trip/:id", s.getTripByID)
}
