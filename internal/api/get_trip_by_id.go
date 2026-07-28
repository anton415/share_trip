package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (s *Server) getTripByID(c *fiber.Ctx) error {
	tripID := c.Params("id")
	if strings.TrimSpace(tripID) == "" {
		return c.Status(fiber.StatusNotFound).JSON(errorResponse{
			Code:    "NOT_FOUND",
			Message: "trip not found",
		})
	}

	trip, err := s.trips.GetTripByID(c.UserContext(), tripID)
	if err != nil {
		return writeFiberServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newTripResponse(trip))
}
