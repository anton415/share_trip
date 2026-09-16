package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"job4j.ru/share-trip/internal/service"
)

func (s *Server) startTrip(c *fiber.Ctx) error {
	clientID, err := clientIDFromClaims(c)
	if err != nil {
		return err
	}
	var request struct {
		TripID uuid.UUID `json:"tripId"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code: "VALIDATION_ERROR", Message: "tripId must be a valid UUID",
		})
	}
	id, err := s.trips.StartTrip(c.UserContext(), service.StartTripCommand{
		TripID:   request.TripID,
		ClientID: clientID,
	})
	if err != nil {
		return writeFiberServiceError(c, err)
	}
	return c.JSON(fiber.Map{"tripId": id})
}
