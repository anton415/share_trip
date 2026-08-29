package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
)

type MoveTripDraftToPublishedRequest struct {
	TripID string `json:"tripId"`
}

type MoveTripDraftToPublishedResponse struct {
	TripID uuid.UUID `json:"tripId"`
}

func (s *Server) moveTripDraftToPublished(c *fiber.Ctx) error {
	clientID, err := clientIDFromClaims(c)
	if err != nil {
		return err
	}

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

	ctx, span := otel.Tracer("trip-api").
		Start(c.UserContext(), "PublishTripHandler")
	defer span.End()

	c.Set("trace-id", span.SpanContext().TraceID().String())

	span.SetAttributes(
		attribute.String("trip_id", tripID.String()),
		attribute.String("client_id", clientID.String()),
	)

	publishedTripID, err := s.trips.PublishTrip(ctx, service.PublishTripCommand{
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
