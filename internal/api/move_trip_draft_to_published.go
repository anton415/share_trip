package api

import (
	"errors"
	"strings"

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
	TripID uuid.UUID `json:"tripId"`
}

func (s *Server) moveTripDraftToPublished(c *fiber.Ctx) error {
	var request MoveTripDraftToPublishedRequest

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "invalid request body",
		})
	}

	if strings.TrimSpace(request.TripID) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "tripId is required",
		})
	}

	tripID, err := uuid.Parse(request.TripID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "tripId must be a valid UUID",
		})
	}

	if strings.TrimSpace(request.ClientID) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "clientId is required",
		})
	}

	clientID, err := uuid.Parse(request.ClientID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "clientId must be a valid UUID",
		})
	}

	publishedTripID, err := s.trips.PublishTrip(c.UserContext(), service.PublishTripCommand{
		TripID:   tripID,
		ClientID: clientID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTripAlreadyPublished) {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return writeFiberServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(MoveTripDraftToPublishedResponse{
		TripID: publishedTripID,
	})
}
