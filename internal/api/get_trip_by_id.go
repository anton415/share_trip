package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"job4j.ru/share-trip/internal/domain"
)

type GetTripByIDResponse struct {
	ID             uuid.UUID         `json:"id"`
	DriverID       uuid.UUID         `json:"driverId"`
	FromPoint      string            `json:"fromPoint"`
	ToPoint        string            `json:"toPoint"`
	DepartureTime  time.Time         `json:"departureTime"`
	AvailableSeats int               `json:"availableSeats"`
	Status         domain.TripStatus `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

func (s *Server) getTripByID(c *fiber.Ctx) error {
	tripID := c.Params("id")
	if strings.TrimSpace(tripID) == "" {
		return c.Status(fiber.StatusNotFound).JSON(errorResponse{
			Code:    "NOT_FOUND",
			Message: "trip not found",
		})
	}

	parsedTripID, err := uuid.Parse(tripID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "tripId must be a valid UUID",
		})
	}

	trip, err := s.trips.GetTripByID(c.UserContext(), parsedTripID)
	if err != nil {
		return writeFiberServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newGetTripByIDResponse(trip))
}
