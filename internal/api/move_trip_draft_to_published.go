package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
)

type MoveTripDraftToPublishedRequest struct {
	TripID   string `json:"tripId"`
	ClientID string `json:"clientId"`
}

type MoveTripDraftToPublishedResponse struct {
	TripID string `json:"tripId"`
}

func (s *Server) moveTripDraftToPublished(c *fiber.Ctx) error {
	var request MoveTripDraftToPublishedRequest

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "invalid request body",
		})
	}

	tripID, tripIDErr := uuid.Parse(request.TripID)
	clientID, clientIDErr := uuid.Parse(request.ClientID)
	if tripIDErr != nil || clientIDErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "invalid request body",
		})
	}

	publishedTripID, err := s.trips.PublishTrip(c.UserContext(), service.PublishTripCommand{
		TripID:   tripID,
		ClientID: clientID,
	})
	if errors.Is(err, domain.ErrTripAlreadyPublished) {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if err != nil {
		return writeFiberServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(MoveTripDraftToPublishedResponse{
		TripID: publishedTripID,
	})
}
